package goproxy

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"log"
	"mars/filterrules"
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
func (h *DefaultDelegate) BeforeRequest(ctx *Context) {
	// Hosts 屏蔽方式 host+ url
	for _, hostlist := range filterrules.Hostlist { // 遍历HOSTS 屏蔽方式
		if gregex.IsMatchString(hostlist, ctx.Req.URL.Host+ctx.Req.URL.Path) {
			// ctx.Req.RemoteAddr = "127.0.0.0"
			ctx.Abort()

		}

	}
	// Req.URL.Path 重写
	for _, list := range filterrules.ReqURLRw { // 遍历Path 重写
		if gregex.IsMatchString(list["url"], ctx.Req.URL.Host+ctx.Req.URL.Path) {

			newlist, err := gregex.ReplaceString(list["target"], list["result"], ctx.Req.URL.Path)
			if err != nil {
				println(err.Error())
			}
			ctx.Req.URL.Path = newlist
		}
	}

	// Req.URL 重定向
	for _, list := range filterrules.ReqURLTo { // 遍历重定向url
		if gregex.IsMatchString(list["url"], ctx.Req.URL.Host+ctx.Req.URL.Path) {
			//{"url": list[0], "target": listRW[0], "result": listRW[1], "urltohost": urltohost, "urltopath": urltopath})

			ctx.Req.URL.Host = list["urltohost"] // 替换host

			ctx.Req.URL.Path = list["urltopath"] // 替换Path
		}
	}
	//// Request Body 新设置
	for _, list := range filterrules.ReqRw { // 遍历重定向url
		if gregex.IsMatchString(list["url"], ctx.Req.URL.Host+ctx.Req.URL.Path) {
			contentType := getContentType(ctx.Req.Header)
			if !IsBinaryBody(contentType) { // 如果不是二进制文件 就执行操作
				bodyBytes, err := ioutil.ReadAll(ctx.Req.Body)
				if err != nil {
					log.Fatalf(" BeforeRequest 读取Body错误: %s", err)
				}

				err = ctx.Req.Body.Close()
				if err != nil {
					log.Fatalf(" BeforeRequest 关闭Body错误: %s", err)
				}

				isGzip := strings.Contains(ctx.Req.Header.Get("Content-Encoding"), "gzip")
				if isGzip {
					ctx.Req.Header.Del("Content-Encoding")
					zipBuf := bytes.NewBuffer(bodyBytes)
					if unGz, err := gzip.NewReader(zipBuf); err == nil {
						bodyBytes, err = ioutil.ReadAll(unGz)
						if err != nil {
							log.Fatalf(" BeforeRequest 读取Body unzip Request错误: %s", err)

						}
						unGz.Close()
					} else {
						log.Fatalf(" BeforeRequest Body unzip Request错误: %s", err)
					}
				}
				// {"url": list[0], "target": listRW[0], "result": listRW[1]})
				bodyBytes = []byte(MarsReplaceString(list["target"], list["result"], bodyBytes))

				ctx.Req.Body = ioutil.NopCloser(bytes.NewReader(bodyBytes))
				bodyLength := len(bodyBytes)
				ctx.Req.ContentLength = int64(bodyLength)
				ctx.Req.Header.Set("Content-Length", strconv.Itoa(bodyLength))
				ct := ctx.Req.Header.Get("Content-Type")
				if !strings.Contains(ct, "utf-8") || !strings.Contains(ct, "UTF-8") {
					ctx.Req.Header.Set("Content-Type", ct+";charset=utf-8")
				}
			}
		}
	}
}

// BeforeResponse 响应发送到客户端前, 修改Header、Body、Status Code
func (h *DefaultDelegate) BeforeResponse(ctx *Context, resp *http.Response, err error) { // 我能个去，写了一半....
	if err != nil {
		log.Fatalf(" BeforeResponse 有err错误: %s", err)

	}
	// resp.Header.Add("X-Request-Id", ctx.Data["req_id"].(string))
	for _, list := range filterrules.RespRw { // 遍历重定向url
		if gregex.IsMatchString(list["url"], ctx.Req.URL.Host+ctx.Req.URL.Path) {
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
				// {"url": list[0], "target": listRW[0], "result": listRW[1]})
				bodyBytes = []byte(MarsReplaceString(list["target"], list["result"], bodyBytes))

				resp.Body = ioutil.NopCloser(bytes.NewReader(bodyBytes))
				bodyLength := len(bodyBytes)
				resp.ContentLength = int64(bodyLength)
				resp.Header.Set("Content-Length", strconv.Itoa(bodyLength))
				ct := resp.Header.Get("Content-Type")
				if !strings.Contains(ct, "utf-8") || !strings.Contains(ct, "UTF-8") {
					resp.Header.Set("Content-Type", ct+";charset=utf-8")
				}
				// log.Println(string(bodyBytes)) // 查看替换成功没
				// ctx.Resp = resp
			}
		}
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

	// content-type: text/html; charset=UTF-8
	// Content-Type: text/html;charset=utf-8
	return contentTypeBinary // 返回是二进制文件
}

// MarsReplaceString 正则替换结果
func MarsReplaceString(target string, result string, body []byte) string {
	s1, err := gregex.ReplaceString(target, result, string(body))
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
