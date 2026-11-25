package user

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	_ "github.com/cloudreve/Cloudreve/v4/ent/runtime"
	entuser "github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster/routes"
	"github.com/cloudreve/Cloudreve/v4/pkg/email"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type registrationTestSettings struct {
	setting.Provider
	activation bool
	groupID    int
}

func (s *registrationTestSettings) EmailActivationEnabled(context.Context) bool { return s.activation }
func (s *registrationTestSettings) DefaultGroup(context.Context) int            { return s.groupID }
func (*registrationTestSettings) SiteURL(context.Context) *url.URL {
	return &url.URL{Scheme: "https", Host: "cloudreve.example"}
}
func (*registrationTestSettings) ActivationEmailTemplate(context.Context) []setting.EmailTemplate {
	return []setting.EmailTemplate{{Title: "Activate", Body: "{{.Url}}"}}
}
func (*registrationTestSettings) SiteBasic(context.Context) *setting.SiteBasic {
	return &setting.SiteBasic{Name: "Registration test"}
}
func (*registrationTestSettings) Logo(context.Context) *setting.Logo { return &setting.Logo{} }

type registrationTestEmail struct {
	email.Driver
	users inventory.UserClient
	auth  auth.Auth
	links []*url.URL
}

func (m *registrationTestEmail) Send(ctx context.Context, to, _, body string) error {
	link, err := url.Parse(html.UnescapeString(body))
	if err != nil {
		return err
	}
	// A separate client must already see the committed credentials when mail is sent.
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	u, err := m.users.GetByEmail(ctx, to)
	if err != nil {
		return err
	}
	if err := m.auth.Check(activationSignContent(u), link.Query().Get("sign")); err != nil {
		return err
	}
	m.links = append(m.links, link)
	return nil
}

type registrationFixture struct {
	dep      dependency.Dep
	users    inventory.UserClient
	settings *registrationTestSettings
	mail     *registrationTestEmail
}

func newRegistrationFixture(t *testing.T) *registrationFixture {
	t.Helper()
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })

	ctx := context.Background()
	client, err := ent.Open("sqlite3", fmt.Sprintf("file:registration-%d?mode=memory&cache=shared", time.Now().UnixNano()))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	require.NoError(t, client.Schema.Create(ctx))
	group, err := client.Group.Create().SetName("test").SetPermissions(&boolset.BooleanSet{}).Save(ctx)
	require.NoError(t, err)
	_, err = client.User.Create().SetEmail("admin@example.com").SetNick("admin").SetGroupID(group.ID).Save(ctx)
	require.NoError(t, err)

	encoder, err := hashid.New("registration-test")
	require.NoError(t, err)
	users := inventory.NewUserClient(client)
	signer := auth.HMACAuth{SecretKey: []byte("registration-test-secret")}
	settings := &registrationTestSettings{activation: true, groupID: group.ID}
	mailer := &registrationTestEmail{users: users, auth: signer}
	dep := dependency.NewDependency(
		dependency.WithUserClient(users),
		dependency.WithSettingProvider(settings),
		dependency.WithEmailClient(mailer),
		dependency.WithHashIDEncoder(encoder),
		dependency.WithGeneralAuth(signer),
	)
	return &registrationFixture{dep: dep, users: users, settings: settings, mail: mailer}
}

func (f *registrationFixture) context(method, target string) *gin.Context {
	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	engine.ContextWithFallback = true
	c.Request = httptest.NewRequest(method, target, nil)
	util.WithValue(c, dependency.DepCtx{}, f.dep)
	return c
}

func (f *registrationFixture) register(address, password string) serializer.Response {
	service := UserRegisterService{UserName: address, Password: password, Language: "en-US"}
	return service.Register(f.context(http.MethodPost, "/api/v4/user"))
}

func (f *registrationFixture) activate(t *testing.T, link *url.URL) serializer.Response {
	t.Helper()
	id := link.Query().Get("id")
	uid, err := f.dep.HashIDEncoder().Decode(id, hashid.UserID)
	require.NoError(t, err)
	c := f.context(http.MethodGet, "/api/v4/user/activate/"+id+"?"+link.RawQuery)
	util.WithValue(c, hashid.ObjectIDCtx{}, uid)
	return ActivateUser(c)
}

func (f *registrationFixture) user(t *testing.T, address string) *ent.User {
	t.Helper()
	u, err := f.users.GetByEmail(context.Background(), address)
	require.NoError(t, err)
	return u
}

func TestRegisterRenewsPendingCredentials(t *testing.T) {
	f := newRegistrationFixture(t)
	require.Equal(t, serializer.CodeNotFullySuccess, f.register("Victim@example.com", "AttackerPass1").Code)
	first := f.user(t, "victim@example.com")
	require.Equal(t, entuser.StatusInactive, first.Status)
	require.NoError(t, inventory.CheckPassword(first, "AttackerPass1"))
	firstLink := f.mail.links[0]

	require.Equal(t, serializer.CodeNotFullySuccess, f.register("VICTIM@example.com", "VictimPass9").Code)
	current := f.user(t, "victim@example.com")
	require.Equal(t, first.ID, current.ID)
	require.Equal(t, entuser.StatusInactive, current.Status)
	require.NoError(t, inventory.CheckPassword(current, "VictimPass9"))
	require.ErrorIs(t, inventory.CheckPassword(current, "AttackerPass1"), inventory.ErrorIncorrectPassword)
	require.Len(t, f.mail.links, 2)

	require.Equal(t, serializer.CodeInvalidSign, f.activate(t, firstLink).Code)
	require.Equal(t, entuser.StatusInactive, f.user(t, "victim@example.com").Status)
	require.Zero(t, f.activate(t, f.mail.links[1]).Code)
	active := f.user(t, "victim@example.com")
	require.Equal(t, entuser.StatusActive, active.Status)
	require.NoError(t, inventory.CheckPassword(active, "VictimPass9"))
	require.Equal(t, serializer.CodeUserCannotActivate, f.activate(t, f.mail.links[1]).Code)

	require.Equal(t, serializer.CodeEmailExisted, f.register("victim@example.com", "AttackerPass2").Code)
	require.Equal(t, active.Password, f.user(t, "victim@example.com").Password)
	require.Len(t, f.mail.links, 2)
}

func TestActivationRejectsSupersededRegistration(t *testing.T) {
	for _, nextPassword := range []string{"AnotherPass3", "VictimPass9"} {
		t.Run(nextPassword, func(t *testing.T) {
			f := newRegistrationFixture(t)
			require.Equal(t, serializer.CodeNotFullySuccess, f.register("victim@example.com", "VictimPass9").Code)
			before := f.user(t, "victim@example.com")
			link := f.mail.links[0]
			require.Equal(t, serializer.CodeNotFullySuccess, f.register("victim@example.com", nextPassword).Code)
			current := f.user(t, "victim@example.com")
			require.NotEqual(t, before.Password, current.Password)

			require.Equal(t, serializer.CodeInvalidSign, f.activate(t, link).Code)
			require.Equal(t, entuser.StatusInactive, f.user(t, "victim@example.com").Status)
			require.Zero(t, f.activate(t, f.mail.links[1]).Code)
			require.NoError(t, inventory.CheckPassword(f.user(t, "victim@example.com"), nextPassword))
		})
	}
}

func TestActivationRejectsInvalidLinks(t *testing.T) {
	f := newRegistrationFixture(t)
	require.Equal(t, serializer.CodeNotFullySuccess, f.register("victim@example.com", "VictimPass9").Code)
	u := f.user(t, "victim@example.com")
	link := f.mail.links[0]
	ttl := time.Now().Add(time.Hour)
	legacyURL, err := auth.SignURI(context.Background(), f.dep.GeneralAuth(),
		routes.MasterUserActivateAPIUrl(f.settings.SiteURL(context.Background()), link.Query().Get("id")).String(), &ttl)
	require.NoError(t, err)

	for _, test := range []struct {
		name string
		sign string
		code int
	}{
		{name: "legacy", sign: legacyURL.Query().Get("sign"), code: serializer.CodeInvalidSign},
		{name: "expired", sign: f.dep.GeneralAuth().Sign(activationSignContent(u), time.Now().Add(-time.Hour).Unix()), code: serializer.CodeSignExpired},
		{name: "tampered", sign: "tampered" + link.Query().Get("sign"), code: serializer.CodeInvalidSign},
	} {
		t.Run(test.name, func(t *testing.T) {
			invalid := *link
			query := invalid.Query()
			query.Set("sign", test.sign)
			invalid.RawQuery = query.Encode()
			response := f.activate(t, &invalid)
			require.Equal(t, test.code, response.Code)
			require.Contains(t, response.Msg, "register again")
			require.Equal(t, entuser.StatusInactive, f.user(t, "victim@example.com").Status)
		})
	}
	require.Zero(t, f.activate(t, link).Code)
}

type activationRaceUsers struct {
	inventory.UserClient
	beforeActivate func(context.Context, *ent.User) error
}

func (u *activationRaceUsers) Activate(ctx context.Context, snapshot *ent.User) (*ent.User, error) {
	if err := u.beforeActivate(ctx, snapshot); err != nil {
		return nil, err
	}
	return u.UserClient.Activate(ctx, snapshot)
}

type registrationTestDep struct {
	dependency.Dep
	users inventory.UserClient
}

func (d registrationTestDep) UserClient() inventory.UserClient { return d.users }

func TestActivationRejectsPasswordChangeAfterVerification(t *testing.T) {
	f := newRegistrationFixture(t)
	require.Equal(t, serializer.CodeNotFullySuccess, f.register("victim@example.com", "VictimPass9").Code)
	called := false
	f.dep = registrationTestDep{Dep: f.dep, users: &activationRaceUsers{
		UserClient: f.users,
		beforeActivate: func(ctx context.Context, snapshot *ent.User) error {
			called = true
			_, err := f.users.UpdateInactivePassword(ctx, snapshot, "AttackerPass2")
			return err
		},
	}}

	require.Equal(t, serializer.CodeInvalidSign, f.activate(t, f.mail.links[0]).Code)
	require.True(t, called)
	current := f.user(t, "victim@example.com")
	require.Equal(t, entuser.StatusInactive, current.Status)
	require.NoError(t, inventory.CheckPassword(current, "AttackerPass2"))
}

func TestInactivePasswordUpdateRejectsStaleState(t *testing.T) {
	for _, change := range []string{"activated", "re-registered"} {
		t.Run(change, func(t *testing.T) {
			f := newRegistrationFixture(t)
			require.Equal(t, serializer.CodeNotFullySuccess, f.register("victim@example.com", "VictimPass9").Code)
			snapshot := f.user(t, "victim@example.com")
			if change == "activated" {
				require.Zero(t, f.activate(t, f.mail.links[0]).Code)
			} else {
				require.Equal(t, serializer.CodeNotFullySuccess, f.register("victim@example.com", "NewOwnerPass9").Code)
			}
			current := f.user(t, "victim@example.com")

			_, err := f.users.UpdateInactivePassword(context.Background(), snapshot, "AttackerPass2")
			require.True(t, ent.IsNotFound(err), "stale update must fail: %v", err)
			unchanged := f.user(t, "victim@example.com")
			require.Equal(t, current.Password, unchanged.Password)
			require.Equal(t, current.Status, unchanged.Status)
		})
	}
}

func TestRegistrationWithActivationDisabled(t *testing.T) {
	f := newRegistrationFixture(t)
	require.Equal(t, serializer.CodeNotFullySuccess, f.register("pending@example.com", "OriginalPass1").Code)
	oldLink := f.mail.links[0]
	f.settings.activation = false

	require.Zero(t, f.register("new@example.com", "NewUserPass9").Code)
	require.Equal(t, entuser.StatusActive, f.user(t, "new@example.com").Status)
	require.Len(t, f.mail.links, 1)

	require.Equal(t, serializer.CodeNotFullySuccess, f.register("pending@example.com", "OwnerPass9").Code)
	require.Equal(t, entuser.StatusInactive, f.user(t, "pending@example.com").Status)
	require.Equal(t, serializer.CodeInvalidSign, f.activate(t, oldLink).Code)
	require.Zero(t, f.activate(t, f.mail.links[1]).Code)
	require.NoError(t, inventory.CheckPassword(f.user(t, "pending@example.com"), "OwnerPass9"))
}
