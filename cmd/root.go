// Package cmd 命令入口
package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mars",
	Short: "规则化的网络调试软件",
}

// Execute 执行命令
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal("命令初始化错误: %s", err)
	}
}
