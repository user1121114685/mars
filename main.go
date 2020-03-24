//go:generate statik -src=web/public -dest=internal -f
// Binary mars HTTP(S)代理
package main

import (
	"mars/cmd"
	"mars/internal/common/version"
)

var (
	// AppVersion 应用版本
	AppVersion string
	// BuildDate 构建日期
	BuildDate string
	// GitCommit 最后提交的git commit
	GitCommit string
)

func main() {
	version.Init(AppVersion, BuildDate, GitCommit) // 提交了个 空的版本号？
	cmd.Execute()                                  // 执行了一个程序 屏幕输出？
}
