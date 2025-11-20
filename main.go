//go:debug rsa1024min=0
package main

import (
	_ "embed"
	"flag"

	"github.com/cloudreve/Cloudreve/v4/cmd"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
)

var (
	isEject    bool
	confPath   string
	scriptName string
)

func init() {
	flag.BoolVar(&util.UseWorkingDir, "use-working-dir", false, "Use working directory, instead of executable directory")
	flag.StringVar(&confPath, "c", util.RelativePath("conf.ini"), "Path to the config file.")
	flag.StringVar(&scriptName, "database-script", "", "Name of database util script.")
	//flag.Parse()

	//staticFS = bootstrap.NewFS(staticZip)
	//bootstrap.Init(confPath, staticFS)
}

func main() {
	cmd.Execute()
	return
	// 关闭数据库连接

	//if scriptName != "" {
	//	// 开始运行助手数据库脚本
	//	bootstrap.RunScript(scriptName)
	//	return
	//}
}
