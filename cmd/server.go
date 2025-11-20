package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/cloudreve/Cloudreve/v4/application"
	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.PersistentFlags().StringVarP(&licenseKey, "license-key", "l", "", "License key of your Cloudreve Pro")
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start a Cloudreve server with the given config file",
	Run: func(cmd *cobra.Command, args []string) {
		dep := dependency.NewDependency(
			dependency.WithConfigPath(confPath),
			dependency.WithProFlag(constants.IsProBool),
			dependency.WithRequiredDbVersion(constants.BackendVersion),
		)
		server := application.NewServer(dep)
		logger := dep.Logger()

		server.PrintBanner()

		// Graceful shutdown after received signal.
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
		go shutdown(sigChan, logger, server)

		if err := server.Start(); err != nil {
			logger.Error("Failed to start server: %s", err)
			os.Exit(1)
		}

		defer func() {
			<-sigChan
		}()
	},
}

func shutdown(sigChan chan os.Signal, logger logging.Logger, server application.Server) {
	sig := <-sigChan
	logger.Info("Signal %s received, shutting down server...", sig)
	server.Close()
	close(sigChan)
}
