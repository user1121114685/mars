package cmd

import (
	"mars/internal/app"
	"mars/internal/app/config"
	"mars/internal/app/inject"
	"mars/internal/common/version"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var Env string
var ConfigFile string

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "启动Mars服务",
	Run: func(cmd *cobra.Command, args []string) {
		log.Info(version.Format())
		viper.BindPFlags(cmd.Flags())
		err := config.CreateConfig(ConfigFile, Env) // 将参数传入config 包，获取后面的参数
		if err != nil {
			log.Print(err)
		}
		conf := config.Conf
		if conf.App.Env.IsDev() {
			log.SetLevel(log.DebugLevel)
		} else {
			log.SetLevel(log.InfoLevel)
		}
		container := inject.NewContainer(conf)
		//	container.Proxy. //goproxy.New(goproxy.WithDelegate(&EventHandler{}))
		app.New(container).Run() // 从这里开始 运行服务
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().StringVarP(&Env, "env", "e", "prod", "dev | prod")                              // 获取传入的参数
	serverCmd.Flags().StringVarP(&ConfigFile, "configFile", "c", "conf/app.toml", "config file path") // 获取传入的参数

}
