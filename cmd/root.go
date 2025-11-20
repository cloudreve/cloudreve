package cmd

import (
	"fmt"
	"os"

	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	confPath   string
	licenseKey string
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&confPath, "conf", "c", util.DataPath("conf.ini"), "Path to the config file")
	rootCmd.PersistentFlags().BoolVarP(&util.UseWorkingDir, "use-working-dir", "w", false, "Use working directory, instead of executable directory")
}

var rootCmd = &cobra.Command{
	Use:   "cloudreve",
	Short: "Cloudreve is a server-side self-hosted cloud storage platform",
	Long: `Self-hosted file management and sharing system, supports multiple storage providers.
Complete documentation is available at https://docs.cloudreve.org/`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}

func Execute() {
	cmd, _, err := rootCmd.Find(os.Args[1:])
	// redirect to default server cmd if no cmd is given
	if err == nil && cmd.Use == rootCmd.Use && cmd.Flags().Parse(os.Args[1:]) != pflag.ErrHelp {
		args := append([]string{"server"}, os.Args[1:]...)
		rootCmd.SetArgs(args)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
