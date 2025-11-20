package application

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
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
	"github.com/cloudreve/Cloudreve/v4/pkg/h3"
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
	dep       dependency.Dep
	logger    logging.Logger
	dbClient  *ent.Client
	config    conf.ConfigProvider
	server    *http.Server
	h3Servers map[string]*h3.H3Server
	kv        cache.Driver
	mailQueue email.Driver
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

	s.server = &http.Server{
		Handler: api,
	}

	s.h3Servers = make(map[string]*h3.H3Server)

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

	api.POST("/api/v4/p2p/signal", s.handleSignal)

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

	// Shutdown http server
	if s.server != nil {
		err := s.server.Shutdown(ctx)
		if err != nil {
			s.logger.Error("Failed to shutdown server: %s", err)
		}
	}
	for laddr, hs := range s.h3Servers {
		err := hs.Shutdown(ctx)
		if err != nil {
			s.logger.Error("Failed to shutdown h3 server: %s %s", laddr, err)
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

func (s *server) handleSignal(c *gin.Context) {
	var req struct {
		Addr string `json:"addr"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.Status(400)
		s.logger.Error("bind req error:%v", err)
		return
	}
	hs, err := h3.NewH3Server()
	if err != nil {
		c.Status(500)
		s.logger.Error("create h3 server error:%v", err)
		return
	}
	hs.Handler = s.server.Handler

	localAddr, pubAddr := hs.GetAddrs()
	c.String(http.StatusOK, pubAddr)
	go func() {
		err := h3.PunchHole(localAddr, req.Addr)
		if err != nil {
			s.logger.Error("punch h3 server error:%v", err)
			return
		}
		s.h3Servers[localAddr] = hs
		err = hs.Serve()
		if err != nil {
			s.logger.Error("punch h3 server error:%v", err)
		}
		delete(s.h3Servers, localAddr)
	}()

}
