package inventory

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/davaccount"
	"github.com/cloudreve/Cloudreve/v4/ent/file"
	"github.com/cloudreve/Cloudreve/v4/ent/passkey"
	"github.com/cloudreve/Cloudreve/v4/ent/schema"
	"github.com/cloudreve/Cloudreve/v4/ent/task"
	"github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/go-webauthn/webauthn/webauthn"
)

type (
	// Ctx keys for eager loading options.
	LoadUserGroup   struct{}
	LoadUserPasskey struct{}

	UserCtx   struct{}
	UserIDCtx struct{}
)

var (
	ErrUserEmailExisted      = errors.New("user email has been registered")
	ErrInactiveUserExisted   = errors.New("email already registered but not activated")
	ErrorUnknownPasswordType = errors.New("unknown password type")
	ErrorIncorrectPassword   = errors.New("incorrect password")
	ErrInsufficientPoints    = errors.New("insufficient points")
)

type (
	UserClient interface {
		TxOperator
		// New creates a new user. If user email registered, existed User will be returned.
		Create(ctx context.Context, args *NewUserArgs) (*ent.User, error)
		// GetByEmail get the user with given email, user status is ignored.
		GetByEmail(ctx context.Context, email string) (*ent.User, error)
		// GetByID get user by its ID, user status is ignored.
		GetByID(ctx context.Context, id int) (*ent.User, error)
		// GetActiveByID get user by its ID, only active user will be returned.
		GetActiveByID(ctx context.Context, id int) (*ent.User, error)
		// SetStatus Set user to given status
		SetStatus(ctx context.Context, u *ent.User, status user.Status) (*ent.User, error)
		// AnonymousUser returns the anonymous user.
		AnonymousUser(ctx context.Context) (*ent.User, error)
		// GetLoginUserByID returns the login user by its ID. It emits some errors and fallback to anonymous user.
		GetLoginUserByID(ctx context.Context, uid int) (*ent.User, error)
		// GetLoginUserByEmail returns the login user by its WebDAV credentials.
		GetActiveByDavAccount(ctx context.Context, email, pwd string) (*ent.User, error)
		// SaveSettings saves user settings.
		SaveSettings(ctx context.Context, u *ent.User) error
		// SearchActive search active users by Email or nickname.
		SearchActive(ctx context.Context, limit int, keyword string) ([]*ent.User, error)
		// ApplyStorageDiff apply storage diff to user.
		ApplyStorageDiff(ctx context.Context, diffs StorageDiff) error
		// UpdateAvatar updates user avatar.
		UpdateAvatar(ctx context.Context, u *ent.User, avatar string) (*ent.User, error)
		// UpdateNickname updates user nickname.
		UpdateNickname(ctx context.Context, u *ent.User, name string) (*ent.User, error)
		// UpdatePassword updates user password.
		UpdatePassword(ctx context.Context, u *ent.User, newPassword string) (*ent.User, error)
		// UpdateTwoFASecret updates user two factor secret.
		UpdateTwoFASecret(ctx context.Context, u *ent.User, secret string) (*ent.User, error)
		// ListPasskeys list user's passkeys.
		ListPasskeys(ctx context.Context, uid int) ([]*ent.Passkey, error)
		// AddPasskey add passkey to user.
		AddPasskey(ctx context.Context, uid int, name string, credential *webauthn.Credential) (*ent.Passkey, error)
		// RemovePasskey remove passkey from user.
		RemovePasskey(ctx context.Context, uid int, keyId string) error
		// MarkPasskeyUsed updates passkey used at.
		MarkPasskeyUsed(ctx context.Context, uid int, keyId string) error
		// CountByTimeRange count users by time range. Will return all records if start or end is nil.
		CountByTimeRange(ctx context.Context, start, end *time.Time) (int, error)
		// ListUsers list users with pagination.
		ListUsers(ctx context.Context, args *ListUserParameters) (*ListUserResult, error)
		// Upsert upserts a user.
		Upsert(ctx context.Context, u *ent.User, password, twoFa string) (*ent.User, error)
		// Delete deletes a user.
		Delete(ctx context.Context, uid int) error
		// CalculateStorage calculate user's storage from scratch and update user's storage.
		CalculateStorage(ctx context.Context, uid int) (int64, error)
	}
	ListUserParameters struct {
		*PaginationArgs
		GroupID int
		Status  user.Status
		Nick    string
		Email   string
	}
	ListUserResult struct {
		*PaginationResults
		Users []*ent.User
	}
)

func NewUserClient(client *ent.Client) UserClient {
	return &userClient{client: client}
}

type userClient struct {
	client *ent.Client
}

type (
	// NewUserArgs args to create a new user
	NewUserArgs struct {
		Email         string
		Nick          string // Optional
		PlainPassword string
		Status        user.Status
		GroupID       int
		Avatar        string // Optional
		Language      string // Optional
	}
	CreateStoragePackArgs struct {
		UserID   int
		Name     string
		Size     int64
		ExpireAt time.Time
	}
)

func (c *userClient) CountByTimeRange(ctx context.Context, start, end *time.Time) (int, error) {
	if start == nil || end == nil {
		return c.client.User.Query().Count(ctx)
	}
	return c.client.User.Query().Where(user.CreatedAtGTE(*start), user.CreatedAtLT(*end)).Count(ctx)
}

func (c *userClient) UpdateNickname(ctx context.Context, u *ent.User, name string) (*ent.User, error) {
	return c.client.User.UpdateOne(u).SetNick(name).Save(ctx)
}

func (c *userClient) UpdateAvatar(ctx context.Context, u *ent.User, avatar string) (*ent.User, error) {
	return c.client.User.UpdateOne(u).SetAvatar(avatar).Save(ctx)
}

func (c *userClient) UpdateTwoFASecret(ctx context.Context, u *ent.User, secret string) (*ent.User, error) {
	if secret == "" {
		return c.client.User.UpdateOne(u).ClearTwoFactorSecret().Save(ctx)
	}
	return c.client.User.UpdateOne(u).SetTwoFactorSecret(secret).Save(ctx)
}

func (c *userClient) UpdatePassword(ctx context.Context, u *ent.User, newPassword string) (*ent.User, error) {
	digest, err := digestPassword(newPassword)
	if err != nil {
		return nil, err
	}

	return c.client.User.UpdateOne(u).SetPassword(digest).Save(ctx)
}

func (c *userClient) SetClient(newClient *ent.Client) TxOperator {
	return &userClient{client: newClient}
}

func (c *userClient) GetClient() *ent.Client {
	return c.client
}

func (c *userClient) ListPasskeys(ctx context.Context, uid int) ([]*ent.Passkey, error) {
	return c.client.Passkey.Query().Where(passkey.UserID(uid)).All(ctx)
}

func (c *userClient) AddPasskey(ctx context.Context, uid int, name string, credential *webauthn.Credential) (*ent.Passkey, error) {
	return c.client.Passkey.Create().
		SetName(name).
		SetCredentialID(base64.StdEncoding.EncodeToString(credential.ID)).
		SetUserID(uid).
		SetCredential(credential).
		Save(ctx)
}

func (c *userClient) RemovePasskey(ctx context.Context, uid int, keyId string) error {
	ctx = schema.SkipSoftDelete(ctx)
	_, err := c.client.Passkey.Delete().Where(passkey.UserID(uid), passkey.CredentialID(keyId)).Exec(ctx)
	return err
}

func (c *userClient) MarkPasskeyUsed(ctx context.Context, uid int, keyId string) error {
	_, err := c.client.Passkey.Update().Where(passkey.UserID(uid), passkey.CredentialID(keyId)).SetUsedAt(time.Now()).Save(ctx)
	return err
}

func (c *userClient) Delete(ctx context.Context, uid int) error {
	// Dav accounts
	if _, err := c.client.DavAccount.Delete().Where(davaccount.OwnerID(uid)).Exec(schema.SkipSoftDelete(ctx)); err != nil {
		return fmt.Errorf("failed to delete dav accounts: %w", err)
	}

	// Passkeys
	if _, err := c.client.Passkey.Delete().Where(passkey.UserID(uid)).Exec(schema.SkipSoftDelete(ctx)); err != nil {
		return fmt.Errorf("failed to delete passkeys: %w", err)
	}

	// Tasks
	if _, err := c.client.Task.Delete().Where(task.UserTasks(uid)).Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete tasks: %w", err)
	}

	return c.client.User.DeleteOneID(uid).Exec(schema.SkipSoftDelete(ctx))
}

func (c *userClient) ApplyStorageDiff(ctx context.Context, diffs StorageDiff) error {
	ae := serializer.NewAggregateError()
	for uid, diff := range diffs {
		// Retry logic for MySQL deadlock (Error 1213)
		// This is a temporary workaround. TODO: optimize storage mutation
		maxRetries := 3
		var lastErr error
		for attempt := 0; attempt < maxRetries; attempt++ {
			if err := c.client.User.Update().Where(user.ID(uid)).AddStorage(diff).Exec(ctx); err != nil {
				lastErr = err
				// Check if it's a MySQL deadlock error (Error 1213)
				if strings.Contains(err.Error(), "Error 1213") && attempt < maxRetries-1 {
					// Wait a bit before retrying with exponential backoff
					time.Sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
					continue
				}
				ae.Add(fmt.Sprintf("%d", uid), fmt.Errorf("failed to apply storage diff for user %d: %w", uid, err))
				break
			}
			// Success, break out of retry loop
			lastErr = nil
			break
		}

		if lastErr != nil {
			ae.Add(fmt.Sprintf("%d", uid), fmt.Errorf("failed to apply storage diff for user %d: %w", uid, lastErr))
		}
	}

	return ae.Aggregate()
}

func (c *userClient) CalculateStorage(ctx context.Context, uid int) (int64, error) {
	var sum int64
	batchSize := 30000
	offset := 0

	for {
		allFiles, err := c.client.File.Query().
			Where(file.HasOwnerWith(user.ID(uid))).
			Where(file.Type(int(types.FileTypeFile))).
			WithEntities().
			Offset(offset).
			Limit(batchSize).
			All(ctx)
		if err != nil {
			return 0, fmt.Errorf("failed to list user files: %w", err)
		}

		if len(allFiles) == 0 {
			break
		}

		for _, file := range allFiles {
			for _, entity := range file.Edges.Entities {
				sum += entity.Size
			}
		}

		offset += batchSize
	}

	if _, err := c.client.User.UpdateOneID(uid).SetStorage(sum).Save(ctx); err != nil {
		return 0, err
	}

	return sum, nil
}

func (c *userClient) SetStatus(ctx context.Context, u *ent.User, status user.Status) (*ent.User, error) {
	return c.client.User.UpdateOne(u).SetStatus(status).Save(ctx)
}

func (c *userClient) Create(ctx context.Context, args *NewUserArgs) (*ent.User, error) {
	// Try to check if there's user with same email.
	if existedUser, err := c.GetByEmail(ctx, args.Email); err == nil {
		if existedUser.Status == user.StatusInactive {
			return existedUser, ErrInactiveUserExisted
		}
		return existedUser, ErrUserEmailExisted
	}

	nick := args.Nick
	if nick == "" {
		nick = strings.Split(args.Email, "@")[0]
	}

	userSetting := &types.UserSetting{VersionRetention: true, VersionRetentionMax: 10}
	query := c.client.User.Create().
		SetEmail(args.Email).
		SetNick(nick).
		SetStatus(args.Status).
		SetGroupID(args.GroupID).
		SetAvatar(args.Avatar)

	if args.PlainPassword != "" {
		pwdDigest, err := digestPassword(args.PlainPassword)
		if err != nil {
			return nil, fmt.Errorf("failed to sha256 password: %w", err)
		}
		query.SetPassword(pwdDigest)
	}

	if args.Language != "" {
		userSetting.Language = args.Language
	}
	query.SetSettings(userSetting)

	// Create user
	newUser, err := query.
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if newUser.ID == 1 {
		// For the first user registered, elevate it to admin group.
		if _, err := newUser.Update().SetGroupID(1).Save(ctx); err != nil {
			return newUser, fmt.Errorf("failed to elevate user to admin: %w", err)
		}
	}
	return newUser, nil
}

func (c *userClient) GetByEmail(ctx context.Context, email string) (*ent.User, error) {
	return withUserEagerLoading(ctx, c.client.User.Query().Where(user.EmailEqualFold(email))).First(ctx)
}

func (c *userClient) GetByID(ctx context.Context, id int) (*ent.User, error) {
	return withUserEagerLoading(ctx, c.client.User.Query().Where(user.ID(id))).First(ctx)
}

func (c *userClient) GetActiveByID(ctx context.Context, id int) (*ent.User, error) {
	return withUserEagerLoading(
		ctx,
		c.client.User.Query().
			Where(user.ID(id)).
			Where(user.StatusEQ(user.StatusActive)),
	).First(ctx)
}

func (c *userClient) GetActiveByDavAccount(ctx context.Context, email, pwd string) (*ent.User, error) {
	ctx = context.WithValue(ctx, LoadUserGroup{}, true)
	return withUserEagerLoading(
		ctx,
		c.client.User.Query().
			Where(user.EmailEqualFold(email)).
			Where(user.StatusEQ(user.StatusActive)).
			WithDavAccounts(func(q *ent.DavAccountQuery) {
				q.Where(davaccount.Password(pwd))
			}),
	).First(ctx)
}

func (c *userClient) GetLoginUserByID(ctx context.Context, uid int) (*ent.User, error) {
	ctx = context.WithValue(ctx, LoadUserGroup{}, true)
	if uid > 0 {
		expectedUser, err := c.GetActiveByID(ctx, uid)
		if err == nil {
			return expectedUser, nil
		}

		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	anonymous, err := c.AnonymousUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to construct anonymous user: %w", err)
	}

	return anonymous, nil
}

func (c *userClient) SearchActive(ctx context.Context, limit int, keyword string) ([]*ent.User, error) {
	ctx = context.WithValue(ctx, LoadUserGroup{}, true)
	return withUserEagerLoading(
		ctx,
		c.client.User.Query().
			Where(user.Or(user.EmailContainsFold(keyword), user.NickContainsFold(keyword))).
			Limit(limit),
	).All(ctx)
}

func (c *userClient) SaveSettings(ctx context.Context, u *ent.User) error {
	return c.client.User.UpdateOne(u).SetSettings(u.Settings).Exec(ctx)
}

// UserFromContext get user from context
func UserFromContext(ctx context.Context) *ent.User {
	u, _ := ctx.Value(UserCtx{}).(*ent.User)
	return u
}

// UserIDFromContext get user id from context.
func UserIDFromContext(ctx context.Context) int {
	uid, ok := ctx.Value(UserIDCtx{}).(int)
	if !ok {
		u := UserFromContext(ctx)
		if u != nil {
			uid = u.ID
		}
	}

	return uid
}

func (c *userClient) AnonymousUser(ctx context.Context) (*ent.User, error) {
	groupClient := NewGroupClient(c.client, "", nil)
	anonymousGroup, err := groupClient.AnonymousGroup(ctx)
	if err != nil {
		return nil, fmt.Errorf("anyonymous group not found: %w", err)
	}

	// TODO: save into cache
	anonymous := &ent.User{
		Settings: &types.UserSetting{},
	}
	anonymous.SetGroup(anonymousGroup)
	return anonymous, nil
}

func (c *userClient) ListUsers(ctx context.Context, args *ListUserParameters) (*ListUserResult, error) {
	query := c.client.User.Query()
	if args.GroupID != 0 {
		query = query.Where(user.GroupUsers(args.GroupID))
	}
	if args.Status != "" {
		query = query.Where(user.StatusEQ(args.Status))
	}
	if args.Nick != "" {
		query = query.Where(user.NickContainsFold(args.Nick))
	}
	if args.Email != "" {
		query = query.Where(user.EmailContainsFold(args.Email))
	}
	query.Order(getUserOrderOption(args)...)

	// Count total items
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}

	users, err := withUserEagerLoading(ctx, query).Limit(args.PageSize).Offset(args.Page * args.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}

	return &ListUserResult{
		PaginationResults: &PaginationResults{
			TotalItems: total,
			Page:       args.Page,
			PageSize:   args.PageSize,
		},
		Users: users,
	}, nil
}

func (c *userClient) Upsert(ctx context.Context, u *ent.User, password, twoFa string) (*ent.User, error) {
	if u.ID == 0 {
		q := c.client.User.Create().
			SetEmail(u.Email).
			SetNick(u.Nick).
			SetAvatar(u.Avatar).
			SetStatus(u.Status).
			SetGroupID(u.GroupUsers).
			SetPassword(u.Password).
			SetSettings(&types.UserSetting{})

		if password != "" {
			pwdDigest, err := digestPassword(password)
			if err != nil {
				return nil, fmt.Errorf("failed to sha256 password: %w", err)
			}
			q.SetPassword(pwdDigest)
		}

		return q.Save(ctx)
	}

	q := c.client.User.UpdateOne(u).
		SetEmail(u.Email).
		SetNick(u.Nick).
		SetAvatar(u.Avatar).
		SetStatus(u.Status).
		SetGroupID(u.GroupUsers)

	if password != "" {
		pwdDigest, err := digestPassword(password)
		if err != nil {
			return nil, fmt.Errorf("failed to sha256 password: %w", err)
		}
		q.SetPassword(pwdDigest)
	}

	if twoFa != "" {
		q.ClearTwoFactorSecret()
	}

	return q.Save(ctx)
}

func getUserOrderOption(args *ListUserParameters) []user.OrderOption {
	orderTerm := getOrderTerm(args.Order)
	switch args.OrderBy {
	case user.FieldNick:
		return []user.OrderOption{user.ByNick(orderTerm), user.ByID(orderTerm)}
	case user.FieldStorage:
		return []user.OrderOption{user.ByStorage(orderTerm), user.ByID(orderTerm)}
	case user.FieldEmail:
		return []user.OrderOption{user.ByEmail(orderTerm), user.ByID(orderTerm)}
	case user.FieldUpdatedAt:
		return []user.OrderOption{user.ByUpdatedAt(orderTerm), user.ByID(orderTerm)}
	default:
		return []user.OrderOption{user.ByID(orderTerm)}
	}
}

// IsAnonymousUser check if given user is anonymous user.
func IsAnonymousUser(u *ent.User) bool {
	return u.ID == 0
}

// CheckPassword 根据明文校验密码
func CheckPassword(u *ent.User, password string) error {
	// 根据存储密码拆分为 Salt 和 Digest
	passwordStore := strings.Split(u.Password, ":")
	if len(passwordStore) != 2 && len(passwordStore) != 3 {
		return ErrorUnknownPasswordType
	}

	// 兼容V2密码，升级后存储格式为: md5:$HASH:$SALT
	if len(passwordStore) == 3 {
		if passwordStore[0] != "md5" {
			return ErrorUnknownPasswordType
		}
		hash := md5.New()
		_, err := hash.Write([]byte(passwordStore[2] + password))
		bs := hex.EncodeToString(hash.Sum(nil))
		if err != nil {
			return err
		}
		if bs != passwordStore[1] {
			return ErrorIncorrectPassword
		}
	}

	//计算 Salt 和密码组合的SHA1摘要
	var hasher hash.Hash
	if len(passwordStore[1]) == 64 {
		hasher = sha256.New()
	} else {
		// Compatible with V3
		hasher = sha1.New()
	}

	_, err := hasher.Write([]byte(password + passwordStore[0]))
	bs := hex.EncodeToString(hasher.Sum(nil))
	if err != nil {
		return err
	}

	if bs != passwordStore[1] {
		return ErrorIncorrectPassword
	}

	return nil
}

func withUserEagerLoading(ctx context.Context, q *ent.UserQuery) *ent.UserQuery {
	if v, ok := ctx.Value(LoadUserGroup{}).(bool); ok && v {
		q.WithGroup(func(gq *ent.GroupQuery) {
			withGroupEagerLoading(ctx, gq)
		})
	}
	if v, ok := ctx.Value(LoadUserPasskey{}).(bool); ok && v {
		q.WithPasskey()
	}
	return q
}

func digestPassword(password string) (string, error) {
	//生成16位 Salt
	salt := util.RandStringRunes(16)

	//计算 Salt 和密码组合的SHA1摘要
	hash := sha256.New()
	_, err := hash.Write([]byte(password + salt))
	bs := hex.EncodeToString(hash.Sum(nil))

	if err != nil {
		return "", err
	}

	//存储 Salt 值和摘要， ":"分割
	return salt + ":" + string(bs), nil
}
