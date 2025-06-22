package dependency

import (
	"context"
	"errors"
	iofs "io/fs"
	"net/url"
	"sync"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/statics"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/credmanager"
	"github.com/cloudreve/Cloudreve/v4/pkg/email"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/mime"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/lock"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/mediameta"
	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/thumb"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-contrib/static"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/robfig/cron/v3"
	"github.com/samber/lo"
	"github.com/ua-parser/uap-go/uaparser"
)

var (
	ErrorConfigPathNotSet = errors.New("config path not set")
)

type (
	// DepCtx defines keys for dependency manager
	DepCtx struct{}
	// ReloadCtx force reload new dependency
	ReloadCtx struct{}
)

// Dep manages all dependencies of the server application. The default implementation is not
// concurrent safe, so all inner deps should be initialized before any goroutine starts.
type Dep interface {
	// ConfigProvider Get a singleton conf.ConfigProvider instance.
	ConfigProvider() conf.ConfigProvider
	// Logger Get a singleton logging.Logger instance.
	Logger() logging.Logger
	// Statics Get a singleton fs.FS instance for embedded static resources.
	Statics() iofs.FS
	// ServerStaticFS Get a singleton static.ServeFileSystem instance for serving static resources.
	ServerStaticFS() static.ServeFileSystem
	// DBClient Get a singleton ent.Client instance for database access.
	DBClient() *ent.Client
	// KV Get a singleton cache.Driver instance for KV store.
	KV() cache.Driver
	// NavigatorStateKV Get a singleton cache.Driver instance for navigator state store. It forces use in-memory
	// map instead of Redis to get better performance for complex nested linked list.
	NavigatorStateKV() cache.Driver
	// SettingClient Get a singleton inventory.SettingClient instance for access DB setting store.
	SettingClient() inventory.SettingClient
	// SettingProvider Get a singleton setting.Provider instance for access setting store in strong type.
	SettingProvider() setting.Provider
	// UserClient Creates a new inventory.UserClient instance for access DB user store.
	UserClient() inventory.UserClient
	// GroupClient Creates a new inventory.GroupClient instance for access DB group store.
	GroupClient() inventory.GroupClient
	// EmailClient Get a singleton email.Driver instance for sending emails.
	EmailClient(ctx context.Context) email.Driver
	// GeneralAuth Get a singleton auth.Auth instance for general authentication.
	GeneralAuth() auth.Auth
	// Shutdown the dependencies gracefully.
	Shutdown(ctx context.Context) error
	// FileClient Creates a new inventory.FileClient instance for access DB file store.
	FileClient() inventory.FileClient
	// NodeClient Creates a new inventory.NodeClient instance for access DB node store.
	NodeClient() inventory.NodeClient
	// DavAccountClient Creates a new inventory.DavAccountClient instance for access DB dav account store.
	DavAccountClient() inventory.DavAccountClient
	// DirectLinkClient Creates a new inventory.DirectLinkClient instance for access DB direct link store.
	DirectLinkClient() inventory.DirectLinkClient
	// HashIDEncoder Get a singleton hashid.Encoder instance for encoding/decoding hashids.
	HashIDEncoder() hashid.Encoder
	// TokenAuth Get a singleton auth.TokenAuth instance for token authentication.
	TokenAuth() auth.TokenAuth
	// LockSystem Get a singleton lock.LockSystem instance for file lock management.
	LockSystem() lock.LockSystem
	// ShareClient Creates a new inventory.ShareClient instance for access DB share store.
	StoragePolicyClient() inventory.StoragePolicyClient
	// RequestClient Creates a new request.Client instance for HTTP requests.
	RequestClient(opts ...request.Option) request.Client
	// ShareClient Creates a new inventory.ShareClient instance for access DB share store.
	ShareClient() inventory.ShareClient
	// TaskClient Creates a new inventory.TaskClient instance for access DB task store.
	TaskClient() inventory.TaskClient
	// ForkWithLogger create a shallow copy of dependency with a new correlated logger, used as per-request dep.
	ForkWithLogger(ctx context.Context, l logging.Logger) context.Context
	// MediaMetaQueue Get a singleton queue.Queue instance for media metadata processing.
	MediaMetaQueue(ctx context.Context) queue.Queue
	// SlaveQueue Get a singleton queue.Queue instance for slave tasks.
	SlaveQueue(ctx context.Context) queue.Queue
	// MediaMetaExtractor Get a singleton mediameta.Extractor instance for media metadata extraction.
	MediaMetaExtractor(ctx context.Context) mediameta.Extractor
	// ThumbPipeline Get a singleton thumb.Generator instance for chained thumbnail generation.
	ThumbPipeline() thumb.Generator
	// ThumbQueue Get a singleton queue.Queue instance for thumbnail generation.
	ThumbQueue(ctx context.Context) queue.Queue
	// EntityRecycleQueue Get a singleton queue.Queue instance for entity recycle.
	EntityRecycleQueue(ctx context.Context) queue.Queue
	// MimeDetector Get a singleton fs.MimeDetector instance for MIME type detection.
	MimeDetector(ctx context.Context) mime.MimeDetector
	// CredManager Get a singleton credmanager.CredManager instance for credential management.
	CredManager() credmanager.CredManager
	// IoIntenseQueue Get a singleton queue.Queue instance for IO intense tasks.
	IoIntenseQueue(ctx context.Context) queue.Queue
	// RemoteDownloadQueue Get a singleton queue.Queue instance for remote download tasks.
	RemoteDownloadQueue(ctx context.Context) queue.Queue
	// NodePool Get a singleton cluster.NodePool instance for node pool management.
	NodePool(ctx context.Context) (cluster.NodePool, error)
	// TaskRegistry Get a singleton queue.TaskRegistry instance for task registration.
	TaskRegistry() queue.TaskRegistry
	// WebAuthn Get a singleton webauthn.WebAuthn instance for WebAuthn authentication.
	WebAuthn(ctx context.Context) (*webauthn.WebAuthn, error)
	// UAParser Get a singleton uaparser.Parser instance for user agent parsing.
	UAParser() *uaparser.Parser
}

type dependency struct {
	configProvider      conf.ConfigProvider
	logger              logging.Logger
	statics             iofs.FS
	serverStaticFS      static.ServeFileSystem
	dbClient            *ent.Client
	rawEntClient        *ent.Client
	kv                  cache.Driver
	navigatorStateKv    cache.Driver
	settingClient       inventory.SettingClient
	fileClient          inventory.FileClient
	shareClient         inventory.ShareClient
	settingProvider     setting.Provider
	userClient          inventory.UserClient
	groupClient         inventory.GroupClient
	storagePolicyClient inventory.StoragePolicyClient
	taskClient          inventory.TaskClient
	nodeClient          inventory.NodeClient
	davAccountClient    inventory.DavAccountClient
	directLinkClient    inventory.DirectLinkClient
	emailClient         email.Driver
	generalAuth         auth.Auth
	hashidEncoder       hashid.Encoder
	tokenAuth           auth.TokenAuth
	lockSystem          lock.LockSystem
	requestClient       request.Client
	ioIntenseQueue      queue.Queue
	thumbQueue          queue.Queue
	mediaMetaQueue      queue.Queue
	entityRecycleQueue  queue.Queue
	slaveQueue          queue.Queue
	remoteDownloadQueue queue.Queue
	ioIntenseQueueTask  queue.Task
	mediaMeta           mediameta.Extractor
	thumbPipeline       thumb.Generator
	mimeDetector        mime.MimeDetector
	credManager         credmanager.CredManager
	nodePool            cluster.NodePool
	taskRegistry        queue.TaskRegistry
	webauthn            *webauthn.WebAuthn
	parser              *uaparser.Parser
	cron                *cron.Cron

	configPath        string
	isPro             bool
	requiredDbVersion string
	licenseKey        string

	// Protects inner deps that can be reloaded at runtime.
	mu sync.Mutex
}

// NewDependency creates a new Dep instance for construct dependencies.
func NewDependency(opts ...Option) Dep {
	d := &dependency{}
	for _, o := range opts {
		o.apply(d)
	}

	return d
}

// FromContext retrieves a Dep instance from context.
func FromContext(ctx context.Context) Dep {
	return ctx.Value(DepCtx{}).(Dep)
}

func (d *dependency) RequestClient(opts ...request.Option) request.Client {
	if d.requestClient != nil {
		return d.requestClient
	}

	return request.NewClient(d.ConfigProvider(), opts...)
}

func (d *dependency) WebAuthn(ctx context.Context) (*webauthn.WebAuthn, error) {
	if d.webauthn != nil {
		return d.webauthn, nil
	}

	settings := d.SettingProvider()
	siteBasic := settings.SiteBasic(ctx)
	wConfig := &webauthn.Config{
		RPDisplayName: siteBasic.Name,
		RPID:          settings.SiteURL(ctx).Hostname(),
		RPOrigins: lo.Map(settings.AllSiteURLs(ctx), func(item *url.URL, index int) string {
			item.Path = ""
			return item.String()
		}), // The origin URLs allowed for WebAuthn requests
	}

	return webauthn.New(wConfig)
}

func (d *dependency) UAParser() *uaparser.Parser {
	if d.parser != nil {
		return d.parser
	}

	d.parser = uaparser.NewFromSaved()
	return d.parser
}

func (d *dependency) ConfigProvider() conf.ConfigProvider {
	if d.configProvider != nil {
		return d.configProvider
	}

	if d.configPath == "" {
		d.panicError(ErrorConfigPathNotSet)
	}

	var err error
	d.configProvider, err = conf.NewIniConfigProvider(d.configPath, logging.NewConsoleLogger(logging.LevelInformational))
	if err != nil {
		d.panicError(err)
	}

	return d.configProvider
}

func (d *dependency) Logger() logging.Logger {
	if d.logger != nil {
		return d.logger
	}

	config := d.ConfigProvider()
	logLevel := logging.LogLevel(config.System().LogLevel)
	if config.System().Debug {
		logLevel = logging.LevelDebug
	}

	d.logger = logging.NewConsoleLogger(logLevel)
	d.logger.Info("Logger initialized with LogLevel=%q.", logLevel)
	return d.logger
}

func (d *dependency) Statics() iofs.FS {
	if d.statics != nil {
		return d.statics
	}

	d.statics = statics.NewStaticFS(d.Logger())
	return d.statics
}

func (d *dependency) ServerStaticFS() static.ServeFileSystem {
	if d.serverStaticFS != nil {
		return d.serverStaticFS
	}

	sfs, err := statics.NewServerStaticFS(d.Logger(), d.Statics(), d.isPro)
	if err != nil {
		d.panicError(err)
	}

	d.serverStaticFS = sfs
	return d.serverStaticFS
}

func (d *dependency) DBClient() *ent.Client {
	if d.dbClient != nil {
		return d.dbClient
	}

	if d.rawEntClient == nil {
		client, err := inventory.NewRawEntClient(d.Logger(), d.ConfigProvider())
		if err != nil {
			d.panicError(err)
		}

		d.rawEntClient = client
	}

	proSuffix := ""
	if d.isPro {
		proSuffix = "-pro"
	}

	client, err := inventory.InitializeDBClient(d.Logger(), d.rawEntClient, d.KV(), d.requiredDbVersion+proSuffix)
	if err != nil {
		d.panicError(err)
	}

	d.dbClient = client
	return d.dbClient
}

func (d *dependency) KV() cache.Driver {
	if d.kv != nil {
		return d.kv
	}

	config := d.ConfigProvider().Redis()
	if config.Server != "" {
		d.kv = cache.NewRedisStore(
			d.Logger(),
			10,
			config,
		)
	} else {
		d.kv = cache.NewMemoStore(util.DataPath(cache.DefaultCacheFile), d.Logger())
	}

	return d.kv
}

func (d *dependency) NavigatorStateKV() cache.Driver {
	if d.navigatorStateKv != nil {
		return d.navigatorStateKv
	}
	d.navigatorStateKv = cache.NewMemoStore("", d.Logger())
	return d.navigatorStateKv
}

func (d *dependency) SettingClient() inventory.SettingClient {
	if d.settingClient != nil {
		return d.settingClient
	}

	d.settingClient = inventory.NewSettingClient(d.DBClient(), d.KV())
	return d.settingClient
}

func (d *dependency) SettingProvider() setting.Provider {
	if d.settingProvider != nil {
		return d.settingProvider
	}

	if d.ConfigProvider().System().Mode == conf.MasterMode {
		// For master mode, setting value will be retrieved in order:
		// Env overwrite -> KV Store -> DB Setting Store
		d.settingProvider = setting.NewProvider(
			setting.NewEnvOverrideStore(
				setting.NewKvSettingStore(d.KV(),
					setting.NewDbSettingStore(d.SettingClient(), nil),
				),
				d.Logger(),
			),
		)
	} else {
		// For slave mode, setting value will be retrieved in order:
		// Env overwrite -> Config file overwrites -> Setting defaults in DB schema
		d.settingProvider = setting.NewProvider(
			setting.NewEnvOverrideStore(
				setting.NewConfSettingStore(d.ConfigProvider(),
					setting.NewDbDefaultStore(nil),
				),
				d.Logger(),
			),
		)
	}

	return d.settingProvider
}

func (d *dependency) UserClient() inventory.UserClient {
	if d.userClient != nil {
		return d.userClient
	}

	return inventory.NewUserClient(d.DBClient())
}

func (d *dependency) GroupClient() inventory.GroupClient {
	if d.groupClient != nil {
		return d.groupClient
	}

	return inventory.NewGroupClient(d.DBClient(), d.ConfigProvider().Database().Type, d.KV())
}

func (d *dependency) NodeClient() inventory.NodeClient {
	if d.nodeClient != nil {
		return d.nodeClient
	}

	return inventory.NewNodeClient(d.DBClient())
}

func (d *dependency) NodePool(ctx context.Context) (cluster.NodePool, error) {
	reload, _ := ctx.Value(ReloadCtx{}).(bool)
	if d.nodePool != nil && !reload {
		return d.nodePool, nil
	}

	if d.ConfigProvider().System().Mode == conf.MasterMode {
		np, err := cluster.NewNodePool(ctx, d.Logger(), d.ConfigProvider(), d.SettingProvider(), d.NodeClient())
		if err != nil {
			return nil, err
		}

		d.nodePool = np
	} else {
		d.nodePool = cluster.NewSlaveDummyNodePool(ctx, d.ConfigProvider(), d.SettingProvider())
	}

	return d.nodePool, nil
}

func (d *dependency) EmailClient(ctx context.Context) email.Driver {
	d.mu.Lock()
	defer d.mu.Unlock()

	if reload, _ := ctx.Value(ReloadCtx{}).(bool); reload || d.emailClient == nil {
		if d.emailClient != nil {
			d.emailClient.Close()
		}
		d.emailClient = email.NewSMTPPool(d.SettingProvider(), d.Logger())
	}

	return d.emailClient
}

func (d *dependency) MimeDetector(ctx context.Context) mime.MimeDetector {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, reload := ctx.Value(ReloadCtx{}).(bool)
	if d.mimeDetector != nil && !reload {
		return d.mimeDetector
	}

	d.mimeDetector = mime.NewMimeDetector(ctx, d.SettingProvider(), d.Logger())
	return d.mimeDetector
}

func (d *dependency) MediaMetaExtractor(ctx context.Context) mediameta.Extractor {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, reload := ctx.Value(ReloadCtx{}).(bool)
	if d.mediaMeta != nil && !reload {
		return d.mediaMeta
	}

	d.mediaMeta = mediameta.NewExtractorManager(ctx, d.SettingProvider(), d.Logger())
	return d.mediaMeta
}

func (d *dependency) ThumbQueue(ctx context.Context) queue.Queue {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, reload := ctx.Value(ReloadCtx{}).(bool)
	if d.thumbQueue != nil && !reload {
		return d.thumbQueue
	}

	if d.thumbQueue != nil {
		d.thumbQueue.Shutdown()
	}

	settings := d.SettingProvider()
	queueSetting := settings.Queue(context.Background(), setting.QueueTypeThumb)
	var (
		t inventory.TaskClient
	)
	if d.ConfigProvider().System().Mode == conf.MasterMode {
		t = d.TaskClient()
	}

	d.thumbQueue = queue.New(d.Logger(), t, nil, d,
		queue.WithBackoffFactor(queueSetting.BackoffFactor),
		queue.WithMaxRetry(queueSetting.MaxRetry),
		queue.WithBackoffMaxDuration(queueSetting.BackoffMaxDuration),
		queue.WithRetryDelay(queueSetting.RetryDelay),
		queue.WithWorkerCount(queueSetting.WorkerNum),
		queue.WithName("ThumbQueue"),
		queue.WithMaxTaskExecution(queueSetting.MaxExecution),
	)
	return d.thumbQueue
}

func (d *dependency) MediaMetaQueue(ctx context.Context) queue.Queue {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, reload := ctx.Value(ReloadCtx{}).(bool)
	if d.mediaMetaQueue != nil && !reload {
		return d.mediaMetaQueue
	}

	if d.mediaMetaQueue != nil {
		d.mediaMetaQueue.Shutdown()
	}

	settings := d.SettingProvider()
	queueSetting := settings.Queue(context.Background(), setting.QueueTypeMediaMeta)

	d.mediaMetaQueue = queue.New(d.Logger(), d.TaskClient(), nil, d,
		queue.WithBackoffFactor(queueSetting.BackoffFactor),
		queue.WithMaxRetry(queueSetting.MaxRetry),
		queue.WithBackoffMaxDuration(queueSetting.BackoffMaxDuration),
		queue.WithRetryDelay(queueSetting.RetryDelay),
		queue.WithWorkerCount(queueSetting.WorkerNum),
		queue.WithName("MediaMetadataQueue"),
		queue.WithMaxTaskExecution(queueSetting.MaxExecution),
		queue.WithResumeTaskType(queue.MediaMetaTaskType),
	)
	return d.mediaMetaQueue
}

func (d *dependency) IoIntenseQueue(ctx context.Context) queue.Queue {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, reload := ctx.Value(ReloadCtx{}).(bool)
	if d.ioIntenseQueue != nil && !reload {
		return d.ioIntenseQueue
	}

	if d.ioIntenseQueue != nil {
		d.ioIntenseQueue.Shutdown()
	}

	settings := d.SettingProvider()
	queueSetting := settings.Queue(context.Background(), setting.QueueTypeIOIntense)

	d.ioIntenseQueue = queue.New(d.Logger(), d.TaskClient(), d.TaskRegistry(), d,
		queue.WithBackoffFactor(queueSetting.BackoffFactor),
		queue.WithMaxRetry(queueSetting.MaxRetry),
		queue.WithBackoffMaxDuration(queueSetting.BackoffMaxDuration),
		queue.WithRetryDelay(queueSetting.RetryDelay),
		queue.WithWorkerCount(queueSetting.WorkerNum),
		queue.WithName("IoIntenseQueue"),
		queue.WithMaxTaskExecution(queueSetting.MaxExecution),
		queue.WithResumeTaskType(queue.CreateArchiveTaskType, queue.ExtractArchiveTaskType, queue.RelocateTaskType, queue.ImportTaskType),
		queue.WithTaskPullInterval(10*time.Second),
	)
	return d.ioIntenseQueue
}

func (d *dependency) RemoteDownloadQueue(ctx context.Context) queue.Queue {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, reload := ctx.Value(ReloadCtx{}).(bool)
	if d.remoteDownloadQueue != nil && !reload {
		return d.remoteDownloadQueue
	}

	if d.remoteDownloadQueue != nil {
		d.remoteDownloadQueue.Shutdown()
	}

	settings := d.SettingProvider()
	queueSetting := settings.Queue(context.Background(), setting.QueueTypeRemoteDownload)

	d.remoteDownloadQueue = queue.New(d.Logger(), d.TaskClient(), d.TaskRegistry(), d,
		queue.WithBackoffFactor(queueSetting.BackoffFactor),
		queue.WithMaxRetry(queueSetting.MaxRetry),
		queue.WithBackoffMaxDuration(queueSetting.BackoffMaxDuration),
		queue.WithRetryDelay(queueSetting.RetryDelay),
		queue.WithWorkerCount(queueSetting.WorkerNum),
		queue.WithName("RemoteDownloadQueue"),
		queue.WithMaxTaskExecution(queueSetting.MaxExecution),
		queue.WithResumeTaskType(queue.RemoteDownloadTaskType),
		queue.WithTaskPullInterval(10*time.Second),
	)
	return d.remoteDownloadQueue
}

func (d *dependency) EntityRecycleQueue(ctx context.Context) queue.Queue {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, reload := ctx.Value(ReloadCtx{}).(bool)
	if d.entityRecycleQueue != nil && !reload {
		return d.entityRecycleQueue
	}

	if d.entityRecycleQueue != nil {
		d.entityRecycleQueue.Shutdown()
	}

	settings := d.SettingProvider()
	queueSetting := settings.Queue(context.Background(), setting.QueueTypeEntityRecycle)

	d.entityRecycleQueue = queue.New(d.Logger(), d.TaskClient(), nil, d,
		queue.WithBackoffFactor(queueSetting.BackoffFactor),
		queue.WithMaxRetry(queueSetting.MaxRetry),
		queue.WithBackoffMaxDuration(queueSetting.BackoffMaxDuration),
		queue.WithRetryDelay(queueSetting.RetryDelay),
		queue.WithWorkerCount(queueSetting.WorkerNum),
		queue.WithName("EntityRecycleQueue"),
		queue.WithMaxTaskExecution(queueSetting.MaxExecution),
		queue.WithResumeTaskType(queue.EntityRecycleRoutineTaskType, queue.ExplicitEntityRecycleTaskType, queue.UploadSentinelCheckTaskType),
		queue.WithTaskPullInterval(10*time.Second),
	)
	return d.entityRecycleQueue
}

func (d *dependency) SlaveQueue(ctx context.Context) queue.Queue {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, reload := ctx.Value(ReloadCtx{}).(bool)
	if d.slaveQueue != nil && !reload {
		return d.slaveQueue
	}

	if d.slaveQueue != nil {
		d.slaveQueue.Shutdown()
	}

	settings := d.SettingProvider()
	queueSetting := settings.Queue(context.Background(), setting.QueueTypeSlave)

	d.slaveQueue = queue.New(d.Logger(), nil, nil, d,
		queue.WithBackoffFactor(queueSetting.BackoffFactor),
		queue.WithMaxRetry(queueSetting.MaxRetry),
		queue.WithBackoffMaxDuration(queueSetting.BackoffMaxDuration),
		queue.WithRetryDelay(queueSetting.RetryDelay),
		queue.WithWorkerCount(queueSetting.WorkerNum),
		queue.WithName("SlaveQueue"),
		queue.WithMaxTaskExecution(queueSetting.MaxExecution),
	)
	return d.slaveQueue
}

func (d *dependency) GeneralAuth() auth.Auth {
	if d.generalAuth != nil {
		return d.generalAuth
	}

	var secretKey string
	if d.ConfigProvider().System().Mode == conf.MasterMode {
		secretKey = d.SettingProvider().SecretKey(context.Background())
	} else {
		secretKey = d.ConfigProvider().Slave().Secret
		if secretKey == "" {
			d.panicError(errors.New("SlaveSecret is not set, please specify it in config file"))
		}
	}

	d.generalAuth = auth.HMACAuth{
		SecretKey: []byte(secretKey),
	}

	return d.generalAuth
}

func (d *dependency) FileClient() inventory.FileClient {
	if d.fileClient != nil {
		return d.fileClient
	}

	return inventory.NewFileClient(d.DBClient(), d.ConfigProvider().Database().Type, d.HashIDEncoder())
}

func (d *dependency) ShareClient() inventory.ShareClient {
	if d.shareClient != nil {
		return d.shareClient
	}

	return inventory.NewShareClient(d.DBClient(), d.ConfigProvider().Database().Type, d.HashIDEncoder())
}

func (d *dependency) TaskClient() inventory.TaskClient {
	if d.taskClient != nil {
		return d.taskClient
	}

	return inventory.NewTaskClient(d.DBClient(), d.ConfigProvider().Database().Type, d.HashIDEncoder())
}

func (d *dependency) DavAccountClient() inventory.DavAccountClient {
	if d.davAccountClient != nil {
		return d.davAccountClient
	}

	return inventory.NewDavAccountClient(d.DBClient(), d.ConfigProvider().Database().Type, d.HashIDEncoder())
}

func (d *dependency) DirectLinkClient() inventory.DirectLinkClient {
	if d.directLinkClient != nil {
		return d.directLinkClient
	}

	return inventory.NewDirectLinkClient(d.DBClient(), d.ConfigProvider().Database().Type, d.HashIDEncoder())
}

func (d *dependency) HashIDEncoder() hashid.Encoder {
	if d.hashidEncoder != nil {
		return d.hashidEncoder
	}

	encoder, err := hashid.New(d.SettingProvider().HashIDSalt(context.Background()))
	if err != nil {
		d.panicError(err)
	}

	d.hashidEncoder = encoder
	return d.hashidEncoder
}

func (d *dependency) CredManager() credmanager.CredManager {
	if d.credManager != nil {
		return d.credManager
	}

	if d.ConfigProvider().System().Mode == conf.MasterMode {
		d.credManager = credmanager.New(d.KV())
	} else {
		d.credManager = credmanager.NewSlaveManager(d.KV(), d.ConfigProvider())
	}
	return d.credManager
}

func (d *dependency) TokenAuth() auth.TokenAuth {
	if d.tokenAuth != nil {
		return d.tokenAuth
	}

	d.tokenAuth = auth.NewTokenAuth(d.HashIDEncoder(), d.SettingProvider(),
		[]byte(d.SettingProvider().SecretKey(context.Background())), d.UserClient(), d.Logger(), d.KV())
	return d.tokenAuth
}

func (d *dependency) LockSystem() lock.LockSystem {
	if d.lockSystem != nil {
		return d.lockSystem
	}

	d.lockSystem = lock.NewMemLS(d.HashIDEncoder(), d.Logger())
	return d.lockSystem
}

func (d *dependency) StoragePolicyClient() inventory.StoragePolicyClient {
	if d.storagePolicyClient != nil {
		return d.storagePolicyClient
	}

	return inventory.NewStoragePolicyClient(d.DBClient(), d.KV())
}

func (d *dependency) ThumbPipeline() thumb.Generator {
	if d.thumbPipeline != nil {
		return d.thumbPipeline
	}

	d.thumbPipeline = thumb.NewPipeline(d.SettingProvider(), d.Logger())
	return d.thumbPipeline
}

func (d *dependency) TaskRegistry() queue.TaskRegistry {
	if d.taskRegistry != nil {
		return d.taskRegistry
	}

	d.taskRegistry = queue.NewTaskRegistry()
	return d.taskRegistry
}

func (d *dependency) Shutdown(ctx context.Context) error {
	d.mu.Lock()

	if d.emailClient != nil {
		d.emailClient.Close()
	}

	wg := sync.WaitGroup{}

	if d.mediaMetaQueue != nil {
		wg.Add(1)
		go func() {
			d.mediaMetaQueue.Shutdown()
			defer wg.Done()
		}()
	}

	if d.thumbQueue != nil {
		wg.Add(1)
		go func() {
			d.thumbQueue.Shutdown()
			defer wg.Done()
		}()
	}

	if d.ioIntenseQueue != nil {
		wg.Add(1)
		go func() {
			d.ioIntenseQueue.Shutdown()
			defer wg.Done()
		}()
	}

	if d.entityRecycleQueue != nil {
		wg.Add(1)
		go func() {
			d.entityRecycleQueue.Shutdown()
			defer wg.Done()
		}()
	}

	if d.slaveQueue != nil {
		wg.Add(1)
		go func() {
			d.slaveQueue.Shutdown()
			defer wg.Done()
		}()
	}

	if d.remoteDownloadQueue != nil {
		wg.Add(1)
		go func() {
			d.remoteDownloadQueue.Shutdown()
			defer wg.Done()
		}()
	}

	d.mu.Unlock()
	wg.Wait()

	return nil
}

func (d *dependency) panicError(err error) {
	if d.logger != nil {
		d.logger.Panic("Fatal error in dependency initialization: %s", err)
	}

	panic(err)
}

func (d *dependency) ForkWithLogger(ctx context.Context, l logging.Logger) context.Context {
	dep := &dependencyCorrelated{
		l:          l,
		dependency: d,
	}
	return context.WithValue(ctx, DepCtx{}, dep)
}

type dependencyCorrelated struct {
	l logging.Logger
	*dependency
}

func (d *dependencyCorrelated) Logger() logging.Logger {
	return d.l
}
