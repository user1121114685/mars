// Package app 应用
package app

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"mars/filterrules"
	"mars/internal/app/inject"
	"mars/internal/app/inspector"
	"mars/shadowsocks"
)

const (
	proxyServerReadTimeout  = 30 * time.Second
	proxyServerWriteTimeout = 5 * time.Second

	inspectorServerReadTimeout  = 30 * time.Second
	inspectorServerWriteTimeout = 5 * time.Second
)

// App 应用
type App struct {
	container *inject.Container
	// goproxy.New(goproxy.WithDelegate(&EventHandler{}))
}

// New 创建应用
func New(container *inject.Container) *App {
	app := &App{
		container: container,
	}

	return app
}

// Run 运行应用
func (app *App) Run() {
	// 初始化规则文件
	filterrules.LoadFilterRules()
	go app.startProxyServer()
	go app.startInspectorServer()
	go shadowsocks.ShadowsocksMain()
	<-app.waitSignal()
}

// 启动代理server
func (app *App) startProxyServer() {
	addr := app.container.Conf.App.ProxyAddr()
	server := &http.Server{
		Addr:         addr,
		Handler:      app.container.Proxy,
		ReadTimeout:  proxyServerReadTimeout,
		WriteTimeout: proxyServerWriteTimeout,
	}
	log.Infof("Proxy server listen on %s", addr)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

// 启动流量审查server
func (app *App) startInspectorServer() {
	inspector.NewRouter(app.container, http.DefaultServeMux).Register()
	addr := app.container.Conf.App.InspectorAddr()
	server := &http.Server{
		Addr:         addr,
		ReadTimeout:  inspectorServerReadTimeout,
		WriteTimeout: inspectorServerWriteTimeout,
	}
	log.Infof("Inspector server listen on %s", addr)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func (app *App) waitSignal() <-chan os.Signal {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)

	return ch
}
