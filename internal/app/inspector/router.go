// Package inspector 流量审查
package inspector

import (
	"io/ioutil"
	"log"
	"net/http"

	"github.com/rakyll/statik/fs"

	"mars/internal/app/inject"
	"mars/internal/app/inspector/controller"
	_ "mars/internal/statik"
)

const staticDir = "/public/" //将这个目录嵌入go 的可执行文件中

// router 路由
type Router struct {
	container *inject.Container
	mux       *http.ServeMux
}

// NewRouter 创建Router
func NewRouter(container *inject.Container, mux *http.ServeMux) *Router {
	r := &Router{
		container: container,
		mux:       mux,
	}

	return r
}

// Register 路由注册
func (r *Router) Register() {
	r.registerStatic()
	c := controller.NewInspector(r.container.WebSocketOutput, r.container.WebSocketSessionOpts)

	r.mux.HandleFunc("/ws", c.WebSocket)
}

func (r *Router) registerStatic() {
	statikFS, err := fs.New() // 将文件嵌入go 的可执行文件中
	if err != nil {
		log.Fatal(err)
	}
	indexFile, err := statikFS.Open("/index.html")
	if err != nil {
		log.Fatal(err)
	}
	indexData, err := ioutil.ReadAll(indexFile)
	if err != nil {
		log.Fatal(err)
	}
	r.mux.Handle(staticDir, http.StripPrefix(staticDir, http.FileServer(statikFS)))
	r.mux.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		rw.Write(indexData)
	})
}
