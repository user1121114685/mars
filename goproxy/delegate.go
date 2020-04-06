package goproxy

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gogf/gf/text/gregex"
)

// Context 代理上下文
type Context struct {
	Req   *http.Request
	Data  map[interface{}]interface{}
	abort bool
	Resp  *http.Response
}

// Abort 中断执行
func (c *Context) Abort() {
	c.abort = true
}

// IsAborted 是否已中断执行
func (c *Context) IsAborted() bool {
	return c.abort
}

type Delegate interface {
	// Connect 收到客户端连接
	Connect(ctx *Context, rw http.ResponseWriter)
	// Auth 代理身份认证
	Auth(ctx *Context, rw http.ResponseWriter)
	// BeforeRequest HTTP请求前 设置X-Forwarded-For, 修改Header、Body
	BeforeRequest(ctx *Context)
	// BeforeResponse 响应发送到客户端前, 修改Header、Body、Status Code
	BeforeResponse(ctx *Context, resp *http.Response, err error)
	// ParentProxy 上级代理
	ParentProxy(*http.Request) (*url.URL, error)
	// Finish 本次请求结束
	Finish(ctx *Context)
	// 记录错误信息
	ErrorLog(err error)
}

var _ Delegate = &DefaultDelegate{}

// DefaultDelegate 默认Handler什么也不做
type DefaultDelegate struct {
	Delegate
}

// Connect 收到客户端连接
func (h *DefaultDelegate) Connect(ctx *Context, rw http.ResponseWriter) {}

// Auth 代理身份认证
func (h *DefaultDelegate) Auth(ctx *Context, rw http.ResponseWriter) {}

// BeforeRequest HTTP请求前 设置X-Forwarded-For, 修改Header、Body
func (h *DefaultDelegate) BeforeRequest(ctx *Context) {}

// BeforeResponse 响应发送到客户端前, 修改Header、Body、Status Code
func (h *DefaultDelegate) BeforeResponse(ctx *Context, resp *http.Response, err error) { // 我能个去，写了一半....
	if err != nil {
		log.Fatalf(" BeforeResponse 有err错误: %s", err)

	}
	// resp.Header.Add("X-Request-Id", ctx.Data["req_id"].(string))

	contentType := getContentType(resp.Header)
	if !IsBinaryBody(contentType) { // 如果不是二进制文件 就执行操作
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf(" BeforeResponse 读取Body错误: %s", err)
		}

		err = resp.Body.Close()
		if err != nil {
			log.Fatalf(" BeforeResponse 关闭Body错误: %s", err)
		}

		isGzip := strings.Contains(resp.Header.Get("Content-Encoding"), "gzip")
		if isGzip {
			resp.Header.Del("Content-Encoding")
			zipBuf := bytes.NewBuffer(bodyBytes)
			if unGz, err := gzip.NewReader(zipBuf); err == nil {
				bodyBytes, err = ioutil.ReadAll(unGz)
				if err != nil {
					log.Fatalf(" BeforeResponse 读取Body unzip response错误: %s", err)

				}
				unGz.Close()
			} else {
				log.Fatalf(" BeforeResponse Body unzip response错误: %s", err)
			}
		}
		bodyBytes = []byte(MarsReplaceString(bodyBytes))

		resp.Body = ioutil.NopCloser(bytes.NewReader(bodyBytes))
		bodyLength := len(bodyBytes)
		resp.ContentLength = int64(bodyLength)
		resp.Header.Set("Content-Length", strconv.Itoa(bodyLength))
		// log.Println(string(bodyBytes)) // 查看替换成功没
		// ctx.Resp = resp
	}

}

// 是否是二进制文件检查
var textMimeTypes = []string{
	"text/xml", "text/html", "text/css", "text/plain", "text/javascript",
	"application/xml", "application/json", "application/javascript", "application/x-www-form-urlencoded",
	"application/x-javascript",
}

const (
	contentTypeBinary = "application/octet-stream" // 二进制
	contentTypePlain  = "text/plain"
)

// IsBinaryBody 检查Body里面是否是二进制
func IsBinaryBody(contentType string) bool {
	for _, item := range textMimeTypes {
		if item == contentType {
			return false
		}
	}

	return true
}

// 获取body类型
func getContentType(h http.Header) string {
	ct := h.Get("Content-Type")
	segments := strings.Split(strings.TrimSpace(ct), ";")
	if len(segments) > 0 && segments[0] != "" {
		return strings.TrimSpace(segments[0])
	}

	return contentTypeBinary // 返回是二进制文件
}

// MarsReplaceString 正则替换结果
func MarsReplaceString(body []byte) string {
	s1, err := gregex.ReplaceString(`.*`, "我的mars成功了！！", string(body))
	if err != nil {
		log.Println(err)
	}
	return s1
}

// ParentProxy 上级代理
func (h *DefaultDelegate) ParentProxy(req *http.Request) (*url.URL, error) {
	return http.ProxyFromEnvironment(req)
}

// Finish 本次请求结束
func (h *DefaultDelegate) Finish(ctx *Context) {}

// ErrorLog 记录错误信息
func (h *DefaultDelegate) ErrorLog(err error) {
	log.Println(err)
}
