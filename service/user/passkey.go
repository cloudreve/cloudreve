package user

import (
	"context"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/activity"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/gofrs/uuid"
	"github.com/samber/lo"
	"strconv"
	"strings"
)

func init() {
	gob.Register(webauthn.SessionData{})
}

type authnUser struct {
	hasher      hashid.Encoder
	u           *ent.User
	credentials []*ent.Passkey
}

func (a *authnUser) WebAuthnID() []byte {
	return []byte(hashid.EncodeUserID(a.hasher, a.u.ID))
}

func (a *authnUser) WebAuthnName() string {
	return a.u.Email
}

func (a *authnUser) WebAuthnDisplayName() string {
	return a.u.Nick
}

func (a *authnUser) WebAuthnCredentials() []webauthn.Credential {
	if a.credentials == nil {
		return nil
	}

	return lo.Map(a.credentials, func(item *ent.Passkey, index int) webauthn.Credential {
		return *item.Credential
	})
}

const (
	authnSessionKey = "authn_session_"
)

func PreparePasskeyLogin(c *gin.Context) (*PreparePasskeyLoginResponse, error) {
	dep := dependency.FromContext(c)
	webAuthn, err := dep.WebAuthn(c)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to initialize WebAuthn", err)
	}

	options, sessionData, err := webAuthn.BeginDiscoverableLogin()
	if err != nil {
		return nil, serializer.NewError(serializer.CodeInitializeAuthn, "Failed to begin registration", err)
	}

	sessionID := uuid.Must(uuid.NewV4()).String()
	if err := dep.KV().Set(fmt.Sprintf("%s%s", authnSessionKey, sessionID), *sessionData, 300); err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to store session data", err)
	}

	return &PreparePasskeyLoginResponse{
		Options:   options,
		SessionID: sessionID,
	}, nil
}

type (
	FinishPasskeyLoginParameterCtx struct{}
	FinishPasskeyLoginService      struct {
		Response  string `json:"response" binding:"required"`
		SessionID string `json:"session_id" binding:"required"`
	}
)

func (s *FinishPasskeyLoginService) FinishPasskeyLogin(c *gin.Context) (*ent.User, error) {
	dep := dependency.FromContext(c)
	kv := dep.KV()
	userClient := dep.UserClient()

	sessionDataRaw, ok := kv.Get(fmt.Sprintf("%s%s", authnSessionKey, s.SessionID))
	if !ok {
		return nil, serializer.NewError(serializer.CodeNotFound, "Session not found", nil)
	}

	_ = kv.Delete(authnSessionKey, s.Response)

	webAuthn, err := dep.WebAuthn(c)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to initialize WebAuthn", err)
	}

	sessionData := sessionDataRaw.(webauthn.SessionData)
	pcc, err := protocol.ParseCredentialRequestResponseBody(strings.NewReader(s.Response))
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "Failed to parse request", err)
	}

	var loginedUser *ent.User
	discoverUserHandle := func(rawID, userHandle []byte) (user webauthn.User, err error) {
		uid, err := dep.HashIDEncoder().Decode(string(userHandle), hashid.UserID)
		if err != nil {
			return nil, err
		}

		ctx := context.WithValue(c, inventory.LoadUserPasskey{}, true)
		ctx = context.WithValue(ctx, inventory.LoadUserGroup{}, true)
		u, err := userClient.GetLoginUserByID(ctx, uid)
		if err != nil {
			return nil, serializer.NewError(serializer.CodeDBError, "Failed to get user", err)
		}

		if inventory.IsAnonymousUser(u) {
			return nil, errors.New("anonymous user")
		}

		loginedUser = u
		return &authnUser{u: u, hasher: dep.HashIDEncoder(), credentials: u.Edges.Passkey}, nil
	}

	credential, err := webAuthn.ValidateDiscoverableLogin(discoverUserHandle, sessionData, pcc)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeWebAuthnCredentialError, "Failed to validate login", err)
	}

	// Find the credential just used
	usedCredentialId := base64.StdEncoding.EncodeToString(credential.ID)
	usedCredential, found := lo.Find(loginedUser.Edges.Passkey, func(item *ent.Passkey) bool {
		return item.CredentialID == usedCredentialId
	})

	if !found {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Passkey login passed but credential used is unknown", nil)
	}

	// Update used at
	if err := userClient.MarkPasskeyUsed(c, loginedUser.ID, usedCredential.CredentialID); err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to update passkey", err)
	}

	return loginedUser, nil
}

func PreparePasskeyRegister(c *gin.Context) (*protocol.CredentialCreation, error) {
	dep := dependency.FromContext(c)
	userClient := dep.UserClient()
	u := inventory.UserFromContext(c)

	existingKeys, err := userClient.ListPasskeys(c, u.ID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list passkeys", err)
	}

	webAuthn, err := dep.WebAuthn(c)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to initialize WebAuthn", err)
	}

	authSelect := protocol.AuthenticatorSelection{
		RequireResidentKey: protocol.ResidentKeyRequired(),
		UserVerification:   protocol.VerificationPreferred,
	}

	options, sessionData, err := webAuthn.BeginRegistration(
		&authnUser{u: u, hasher: dep.HashIDEncoder()},
		webauthn.WithAuthenticatorSelection(authSelect),
		webauthn.WithExclusions(lo.Map(existingKeys, func(item *ent.Passkey, index int) protocol.CredentialDescriptor {
			return protocol.CredentialDescriptor{
				Type:            protocol.PublicKeyCredentialType,
				CredentialID:    item.Credential.ID,
				Transport:       item.Credential.Transport,
				AttestationType: item.Credential.AttestationType,
			}
		})),
	)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeInitializeAuthn, "Failed to begin registration", err)
	}

	if err := dep.KV().Set(fmt.Sprintf("%s%d", authnSessionKey, u.ID), *sessionData, 300); err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to store session data", err)
	}

	return options, nil
}

type (
	FinishPasskeyRegisterParameterCtx struct{}
	FinishPasskeyRegisterService      struct {
		Response string `json:"response" binding:"required"`
		Name     string `json:"name" binding:"required"`
		UA       string `json:"ua" binding:"required"`
	}
)

func (s *FinishPasskeyRegisterService) FinishPasskeyRegister(c *gin.Context) (*Passkey, error) {
	dep := dependency.FromContext(c)
	kv := dep.KV()
	u := inventory.UserFromContext(c)

	sessionDataRaw, ok := kv.Get(fmt.Sprintf("%s%d", authnSessionKey, u.ID))
	if !ok {
		return nil, serializer.NewError(serializer.CodeNotFound, "Session not found", nil)
	}

	_ = kv.Delete(authnSessionKey, strconv.Itoa(u.ID))

	webAuthn, err := dep.WebAuthn(c)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to initialize WebAuthn", err)
	}

	sessionData := sessionDataRaw.(webauthn.SessionData)
	pcc, err := protocol.ParseCredentialCreationResponseBody(strings.NewReader(s.Response))
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "Failed to parse request", err)
	}

	credential, err := webAuthn.CreateCredential(&authnUser{u: u, hasher: dep.HashIDEncoder()}, sessionData, pcc)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeWebAuthnCredentialError, "Failed to finish registration", err)
	}

	client := dep.UAParser().Parse(s.UA)
	name := util.Replace(map[string]string{
		"{os}":      client.Os.Family,
		"{browser}": client.UserAgent.Family,
	}, s.Name)

	passkey, err := dep.UserClient().AddPasskey(c, u.ID, name, credential)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to add passkey", err)
	}
	activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventAddPasskey,
		activity.Extra(map[string]any{"name": name}))

	res := BuildPasskey(passkey)
	return &res, nil
}

type (
	DeletePasskeyService struct {
		ID string `form:"id" binding:"required"`
	}
	DeletePasskeyParameterCtx struct{}
)

func (s *DeletePasskeyService) DeletePasskey(c *gin.Context) error {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)
	userClient := dep.UserClient()

	existingKeys, err := userClient.ListPasskeys(c, u.ID)
	if err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to list passkeys", err)
	}

	var existing *ent.Passkey
	for _, key := range existingKeys {
		if key.CredentialID == s.ID {
			existing = key
			break
		}
	}

	if existing == nil {
		return serializer.NewError(serializer.CodeNotFound, "Passkey not found", nil)
	}

	if err := userClient.RemovePasskey(c, u.ID, s.ID); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to delete passkey", err)
	}
	activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventRemovePasskey,
		activity.Extra(map[string]any{"credential_id": s.ID}))

	return nil
}
