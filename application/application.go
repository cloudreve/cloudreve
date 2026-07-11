package application

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/crontab"
	"github.com/cloudreve/Cloudreve/v4/pkg/email"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver/onedrive"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/cloudreve/Cloudreve/v4/routers"
	"github.com/gin-gonic/gin"
)

type Server interface {
	// Start starts the Cloudreve server.
	Start() error
	PrintBanner()
	Close()
}

// NewServer constructs a new Cloudreve server instance with given dependency.
func NewServer(dep dependency.Dep) Server {
	return &server{
		dep:    dep,
		logger: dep.Logger(),
		config: dep.ConfigProvider(),
	}
}

type server struct {
	dep         dependency.Dep
	logger      logging.Logger
	dbClient    *ent.Client
	config      conf.ConfigProvider
	server      *http.Server
	pprofServer *http.Server
	kv          cache.Driver
	mailQueue   email.Driver
}

func (s *server) PrintBanner() {
	fmt.Print(`
   ___ _                 _                    
  / __\ | ___  _   _  __| |_ __ _____   _____ 
 / /  | |/ _ \| | | |/ _  | '__/ _ \ \ / / _ \	
/ /___| | (_) | |_| | (_| | | |  __/\ V /  __/
\____/|_|\___/ \__,_|\__,_|_|  \___| \_/ \___|

   V` + constants.BackendVersion + `  Commit #` + constants.LastCommit + `  Pro=` + constants.IsPro + `
================================================

`)
}

func (s *server) Start() error {
	// Debug 关闭时，切换为生产模式
	if !s.config.System().Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	s.kv = s.dep.KV()
	// delete all cached settings
	_ = s.kv.Delete(setting.KvSettingPrefix)
	if memKv, ok := s.kv.(*cache.MemoStore); ok {
		memKv.GarbageCollect(s.logger)
	}

	// TODO: make sure redis is connected in dep before user traffic.
	if s.config.System().Mode == conf.MasterMode {
		s.dbClient = s.dep.DBClient()
		// TODO: make sure all dep is initialized before server start.
		s.dep.LockSystem()
		s.dep.UAParser()

		// Initialize OneDrive credentials
		credentials, err := onedrive.RetrieveOneDriveCredentials(context.Background(), s.dep.StoragePolicyClient())
		if err != nil {
			return fmt.Errorf("faield to retrieve OneDrive credentials for CredManager: %w", err)
		}
		if err := s.dep.CredManager().Upsert(context.Background(), credentials...); err != nil {
			return fmt.Errorf("failed to upsert OneDrive credentials to CredManager: %w", err)
		}
		crontab.Register(setting.CronTypeOauthCredRefresh, func(ctx context.Context) {
			dep := dependency.FromContext(ctx)
			cred := dep.CredManager()
			cred.RefreshAll(ctx)
		})

		// Initialize email queue before user traffic starts.
		_ = s.dep.EmailClient(context.Background())

		// Start all queues
		s.dep.MediaMetaQueue(context.Background()).Start()
		s.dep.EntityRecycleQueue(context.Background()).Start()
		s.dep.IoIntenseQueue(context.Background()).Start()
		s.dep.RemoteDownloadQueue(context.Background()).Start()
		s.dep.MediaProcessQueue(context.Background()).Start()
		s.dep.MediaVideoQueue(context.Background()).Start()

		// Start cron jobs
		c, err := crontab.NewCron(context.Background(), s.dep)
		if err != nil {
			return err
		}
		c.Start()

		// Start node pool
		if _, err := s.dep.NodePool(context.Background()); err != nil {
			return err
		}
	} else {
		s.dep.SlaveQueue(context.Background()).Start()
	}
	s.dep.ThumbQueue(context.Background()).Start()

	api := routers.InitRouter(s.dep)
	api.TrustedPlatform = s.config.System().ProxyHeader
	s.server = &http.Server{Handler: api}

	// Start pprof server if configured
	if pprofAddr := s.config.System().Pprof; pprofAddr != "" {
		s.pprofServer = &http.Server{
			Addr:    pprofAddr,
			Handler: http.DefaultServeMux,
		}
		go func() {
			s.logger.Info("pprof server listening on %q", pprofAddr)
			if err := s.pprofServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				s.logger.Error("pprof server error: %s", err)
			}
		}()
	}

	// 如果启用了SSL
	if s.config.SSL().CertPath != "" {
		s.logger.Info("Listening to %q", s.config.SSL().Listen)
		s.server.Addr = s.config.SSL().Listen
		if err := s.server.ListenAndServeTLS(s.config.SSL().CertPath, s.config.SSL().KeyPath); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("failed to listen to %q: %w", s.config.SSL().Listen, err)
		}

		return nil
	}

	// 如果启用了Unix
	if s.config.Unix().Listen != "" {
		// delete socket file before listening
		if _, err := os.Stat(s.config.Unix().Listen); err == nil {
			if err = os.Remove(s.config.Unix().Listen); err != nil {
				return fmt.Errorf("failed to delete socket file %q: %w", s.config.Unix().Listen, err)
			}
		}

		s.logger.Info("Listening to %q", s.config.Unix().Listen)
		if err := s.runUnix(s.server); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("failed to listen to %q: %w", s.config.Unix().Listen, err)
		}

		return nil
	}

	s.logger.Info("Listening to %q", s.config.System().Listen)
	s.server.Addr = s.config.System().Listen
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to listen to %q: %w", s.config.System().Listen, err)
	}
	return nil
}

func (s *server) Close() {
	if s.dbClient != nil {
		s.logger.Info("Shutting down database connection...")
		if err := s.dbClient.Close(); err != nil {
			s.logger.Error("Failed to close database connection: %s", err)
		}
	}

	ctx := context.Background()
	if conf.SystemConfig.GracePeriod != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(s.config.System().GracePeriod)*time.Second)
		defer cancel()
	}

	s.dep.EventHub().Close()

	// Shutdown http server
	if s.server != nil {
		err := s.server.Shutdown(ctx)
		if err != nil {
			s.logger.Error("Failed to shutdown server: %s", err)
		}
	}

	// Shutdown pprof server
	if s.pprofServer != nil {
		if err := s.pprofServer.Shutdown(ctx); err != nil {
			s.logger.Error("Failed to shutdown pprof server: %s", err)
		}
	}

	if s.kv != nil {
		if err := s.kv.Persist(util.DataPath(cache.DefaultCacheFile)); err != nil {
			s.logger.Warning("Failed to persist cache: %s", err)
		}
	}

	if err := s.dep.Shutdown(ctx); err != nil {
		s.logger.Warning("Failed to shutdown dependency manager: %s", err)
	}
}

func (s *server) runUnix(server *http.Server) error {
	listener, err := net.Listen("unix", s.config.Unix().Listen)
	if err != nil {
		return err
	}

	defer listener.Close()
	defer os.Remove(s.config.Unix().Listen)

	if conf.UnixConfig.Perm > 0 {
		err = os.Chmod(conf.UnixConfig.Listen, os.FileMode(s.config.Unix().Perm))
		if err != nil {
			s.logger.Warning(
				"Failed to set permission to %q for socket file %q: %s",
				s.config.Unix().Perm,
				s.config.Unix().Listen,
				err,
			)
		}
	}

	return server.Serve(listener)
}
