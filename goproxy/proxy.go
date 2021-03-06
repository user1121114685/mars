// Package goproxy HTTP(S)代理, 支持中间人代理解密HTTPS数据
package goproxy

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"mars/filterrules"
	"mars/goproxy/cert"

	"github.com/gogf/gf/text/gregex"
)

const (
	// 连接目标服务器超时时间
	defaultTargetConnectTimeout = 5 * time.Second
	// 目标服务器读写超时时间
	defaultTargetReadWriteTimeout = 30 * time.Second
	// 客户端读写超时时间
	defaultClientReadWriteTimeout = 30 * time.Second
)

// tunnelEstablishedResponseLine 隧道连接成功响应行
var tunnelEstablishedResponseLine = []byte("HTTP/1.1 200 Connection established\r\n\r\n")

var badGateway = []byte(fmt.Sprintf("HTTP/1.1 %d %s\r\n\r\n", http.StatusBadGateway, http.StatusText(http.StatusBadGateway)))

// 生成隧道建立请求行
func makeTunnelRequestLine(addr string) string {
	return fmt.Sprintf("CONNECT %s HTTP/1.1\r\n\r\n", addr)
}

type options struct {
	disableKeepAlive bool
	delegate         Delegate
	decryptHTTPS     bool
	certCache        cert.Cache
	transport        *http.Transport
}

type Option func(*options)

// WithDisableKeepAlive 连接是否重用
func WithDisableKeepAlive(disableKeepAlive bool) Option {
	return func(opt *options) {
		opt.disableKeepAlive = disableKeepAlive
	}
}

// WithDelegate 设置委托类
func WithDelegate(delegate Delegate) Option {
	return func(opt *options) {
		opt.delegate = delegate
	}
}

// WithTransport 自定义http transport
func WithTransport(t *http.Transport) Option {
	return func(opt *options) {
		opt.transport = t
	}
}

// WithDecryptHTTPS 中间人代理, 解密HTTPS, 需实现证书缓存接口
func WithDecryptHTTPS(c cert.Cache) Option {
	return func(opt *options) {
		opt.decryptHTTPS = true
		opt.certCache = c
	}
}

// New 创建proxy实例
func New(opt ...Option) *Proxy {
	opts := &options{}
	for _, o := range opt {
		o(opts)
	}

	if opts.delegate == nil {
		opts.delegate = &DefaultDelegate{}
	}
	// delegateMars := &DefaultDelegate{}
	if opts.transport == nil {
		opts.transport = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
	}

	p := &Proxy{}
	p.delegate = opts.delegate
	p.delegateMars = &DefaultDelegate{}
	p.decryptHTTPS = opts.decryptHTTPS
	if p.decryptHTTPS {
		p.cert = &cert.Certificate{
			Cache: opts.certCache,
		}
	}
	p.transport = opts.transport
	p.transport.DisableKeepAlives = opts.disableKeepAlive
	p.transport.Proxy = p.delegate.ParentProxy

	return p
}

// Proxy 实现了http.Handler接口
type Proxy struct {
	delegate      Delegate // 在这里可以稍作修改 支持2个delegate
	clientConnNum int32
	decryptHTTPS  bool // 是否解密 SSl证书
	cert          *cert.Certificate
	transport     *http.Transport
	delegateMars  Delegate // 专门用来修改数据
}

var _ http.Handler = &Proxy{}

// ServeHTTP 实现了http.Handler接口
func (p *Proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	if req.URL.Host == "" {
		req.URL.Host = req.Host
	}
	// 白名单放行
	pass := 0
	for _, whitelist := range filterrules.Whitelist { // 遍历白名单
		if gregex.IsMatchString(whitelist, req.Host) {
			pass = 1
		}

	}
	for _, blacklist := range filterrules.Blacklist { // 遍历需要过滤的名单
		if gregex.IsMatchString(blacklist, req.Host) {
			pass = 2
		}

	}
	atomic.AddInt32(&p.clientConnNum, 1)
	defer func() {
		atomic.AddInt32(&p.clientConnNum, -1)
	}()
	ctx := &Context{
		Req:  req,
		Data: make(map[interface{}]interface{}),
	}
	defer p.delegate.Finish(ctx)
	p.delegate.Connect(ctx, rw)
	if ctx.abort {
		return
	}
	p.delegate.Auth(ctx, rw)
	if ctx.abort {
		return
	}

	switch {
	case pass == 1: // 白名单
		pass = 0
		p.forwardTunnel(ctx, rw)
	case pass == 2 || ctx.Req.Method == http.MethodConnect && p.decryptHTTPS:
		pass = 0
		p.forwardHTTPS(ctx, rw)
	case pass == 0 || ctx.Req.Method == http.MethodConnect:
		p.forwardTunnel(ctx, rw)
	default:
		p.forwardHTTP(ctx, rw)
	}

}

// ClientConnNum 获取客户端连接数
func (p *Proxy) ClientConnNum() int32 {
	return atomic.LoadInt32(&p.clientConnNum)
}

// DoRequest 执行HTTP请求，并调用responseFunc处理response
func (p *Proxy) DoRequest(ctx *Context, responseFunc func(*http.Response, error)) {

	if ctx.Data == nil {
		ctx.Data = make(map[interface{}]interface{})
	}
	p.delegateMars.BeforeRequest(ctx)
	p.delegate.BeforeRequest(ctx) // 将修改好的内容传送到web 端口
	if ctx.abort {
		return
	}
	newReq := new(http.Request)
	*newReq = *ctx.Req
	newReq.Header = CloneHeader(newReq.Header)
	removeConnectionHeaders(newReq.Header)
	for _, item := range hopHeaders {
		if newReq.Header.Get(item) != "" {
			newReq.Header.Del(item)
		}
	}

	//  Request Headers 删除
	for _, list := range filterrules.ReqDel { // 遍历重定向url
		if gregex.IsMatchString(list["url"], ctx.Req.URL.Host+ctx.Req.URL.Path) {
			//{"url": list[0], "headerName": list[1]})
			newReq.Header.Del(list["headerName"])

		}
	}

	//  Request Headers 追加设置
	for _, list := range filterrules.ReqOriSet { // 遍历重定向url
		if gregex.IsMatchString(list["url"], ctx.Req.URL.Host+ctx.Req.URL.Path) {
			//{"url": list[0], "target": listRW[0], "result": listRW[1]}
			ori := newReq.Header.Get(list["target"])
			newReq.Header.Set(list["target"], ori+";"+list["result"])

		}
	}

	//  Request Headers 追加设置
	for _, list := range filterrules.ReqNewSet { // 遍历重定向url
		if gregex.IsMatchString(list["url"], ctx.Req.URL.Host+ctx.Req.URL.Path) {
			//{"url": list[0], "target": listRW[0], "result": listRW[1]}

			newReq.Header.Set(list["target"], list["result"])

		}
	}

	resp, err := p.transport.RoundTrip(newReq)

	p.delegateMars.BeforeResponse(ctx, resp, err) // 这里修改传回内容
	p.delegate.BeforeResponse(ctx, resp, err)     // 将修改好的内容传送到web 端口
	if ctx.abort {
		return
	}
	if err == nil {
		removeConnectionHeaders(resp.Header)
		for _, h := range hopHeaders {
			resp.Header.Del(h)
		}
	}

	//  Response Headers 删除
	for _, list := range filterrules.RespDel { // 遍历重定向url
		if gregex.IsMatchString(list["url"], ctx.Req.URL.Host+ctx.Req.URL.Path) {
			//{"url": list[0], "headerName": list[1]})
			resp.Header.Del(list["headerName"])

		}
	}

	//  Response Headers 追加设置
	for _, list := range filterrules.RespOriSet { // 遍历重定向url
		if gregex.IsMatchString(list["url"], ctx.Req.URL.Host+ctx.Req.URL.Path) {
			//{"url": list[0], "target": listRW[0], "result": listRW[1]}
			ori := resp.Header.Get(list["target"])
			resp.Header.Set(list["target"], ori+";"+list["result"])

		}
	}

	//  Response Headers 追加设置
	for _, list := range filterrules.RespNewSet { // 遍历重定向url
		if gregex.IsMatchString(list["url"], ctx.Req.URL.Host+ctx.Req.URL.Path) {
			//{"url": list[0], "target": listRW[0], "result": listRW[1]}

			resp.Header.Set(list["target"], list["result"])

		}
	}
	responseFunc(resp, err)
}

// HTTP转发
func (p *Proxy) forwardHTTP(ctx *Context, rw http.ResponseWriter) {
	ctx.Req.URL.Scheme = "http"
	p.DoRequest(ctx, func(resp *http.Response, err error) {
		if err != nil {
			p.delegate.ErrorLog(fmt.Errorf("%s - HTTP请求错误: , 错误: %s", ctx.Req.URL, err))
			rw.WriteHeader(http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		CopyHeader(rw.Header(), resp.Header)
		rw.WriteHeader(resp.StatusCode)
		io.Copy(rw, resp.Body)
	})
}

// HTTPS转发
func (p *Proxy) forwardHTTPS(ctx *Context, rw http.ResponseWriter) {
	clientConn, err := hijacker(rw)
	if err != nil {
		p.delegate.ErrorLog(err)
		rw.WriteHeader(http.StatusBadGateway)
		return
	}
	defer clientConn.Close()
	_, err = clientConn.Write(tunnelEstablishedResponseLine) // 修改隧道响应成功  HTTP/1.1 200 Connection established
	if err != nil {
		p.delegate.ErrorLog(fmt.Errorf("%s - HTTPS解密, 通知客户端隧道已连接失败, %s", ctx.Req.URL.Host, err))
		return
	}
	// 创建ssl 解密证书
	tlsConfig, err := p.cert.Generate(ctx.Req.URL.Host)
	if err != nil {
		p.delegate.ErrorLog(fmt.Errorf("%s - HTTPS解密, 生成证书失败: %s", ctx.Req.URL.Host, err))
		rw.WriteHeader(http.StatusBadGateway)
		return
	}
	// tls.Server使用conn作为下层传输接口返回一个TLS连接的服务端侧。配置参数config必须是非nil的且必须含有至少一个证书。
	tlsClientConn := tls.Server(clientConn, tlsConfig)
	tlsClientConn.SetDeadline(time.Now().Add(defaultClientReadWriteTimeout)) // SetDeadline设置该连接的读写操作绝对期限。t为Time零值表示不设置超时。在一次Write/Read方法超时后，TLS连接状态会被破坏，之后所有的读写操作都会返回同一错误。
	defer tlsClientConn.Close()

	if err := tlsClientConn.Handshake(); err != nil {
		p.delegate.ErrorLog(fmt.Errorf("%s - HTTPS解密, 握手失败: %s", ctx.Req.URL.Host, err))
		return
	}

	buf := bufio.NewReader(tlsClientConn) // 读取ssl conn的内容，

	tlsReq, err := http.ReadRequest(buf) // 读取 Request // 修改http 头部 的内容就从此开始 的内容就从此开始

	if err != nil {
		if err != io.EOF {
			p.delegate.ErrorLog(fmt.Errorf("%s - HTTPS解密, 读取客户端请求失败: %s", ctx.Req.URL.Host, err))
		}
		return
	}

	// 给 tlsReq的几个结果赋值
	tlsReq.RemoteAddr = ctx.Req.RemoteAddr
	tlsReq.URL.Scheme = "https"
	tlsReq.URL.Host = tlsReq.Host

	ctx.Req = tlsReq

	p.DoRequest(ctx, func(resp *http.Response, err error) {
		if err != nil {
			p.delegate.ErrorLog(fmt.Errorf("%s - HTTPS解密, 请求错误: %s", ctx.Req.URL, err))
			tlsClientConn.Write(badGateway)
			return
		}
		err = resp.Write(tlsClientConn)
		if err != nil {
			p.delegate.ErrorLog(fmt.Errorf("%s - HTTPS解密, response写入客户端失败, %s", ctx.Req.URL, err))
		}
		resp.Body.Close()
	})
}

// 隧道转发
func (p *Proxy) forwardTunnel(ctx *Context, rw http.ResponseWriter) {
	clientConn, err := hijacker(rw)
	if err != nil {
		p.delegate.ErrorLog(err)
		rw.WriteHeader(http.StatusBadGateway)
		return
	}
	defer clientConn.Close()
	parentProxyURL, err := p.delegate.ParentProxy(ctx.Req)
	if err != nil {
		p.delegate.ErrorLog(fmt.Errorf("%s - 解析代理地址错误: %s", ctx.Req.URL.Host, err))
		rw.WriteHeader(http.StatusBadGateway)
		return
	}
	targetAddr := ctx.Req.URL.Host
	if parentProxyURL != nil {
		targetAddr = parentProxyURL.Host
	}

	targetConn, err := net.DialTimeout("tcp", targetAddr, defaultTargetConnectTimeout)
	if err != nil {
		p.delegate.ErrorLog(fmt.Errorf("%s - 隧道转发连接目标服务器失败: %s", ctx.Req.URL.Host, err))
		rw.WriteHeader(http.StatusBadGateway)
		return
	}
	defer targetConn.Close()

	clientConn.SetDeadline(time.Now().Add(defaultClientReadWriteTimeout))
	targetConn.SetDeadline(time.Now().Add(defaultTargetReadWriteTimeout))
	if parentProxyURL == nil {
		_, err = clientConn.Write(tunnelEstablishedResponseLine)
		if err != nil {
			p.delegate.ErrorLog(fmt.Errorf("%s - 隧道连接成功,通知客户端错误: %s", ctx.Req.URL.Host, err))
			return
		}
	} else {
		tunnelRequestLine := makeTunnelRequestLine(ctx.Req.URL.Host)
		targetConn.Write([]byte(tunnelRequestLine))
	}

	p.transfer(clientConn, targetConn)
}

// 双向转发
func (p *Proxy) transfer(src net.Conn, dst net.Conn) {
	go func() {
		io.Copy(src, dst)
		src.Close()
		dst.Close()
	}()

	io.Copy(dst, src)
	dst.Close()
	src.Close()
}

// 获取底层连接
func hijacker(rw http.ResponseWriter) (net.Conn, error) {
	hijacker, ok := rw.(http.Hijacker)
	if !ok {
		return nil, fmt.Errorf("web server不支持Hijacker")
	}
	conn, _, err := hijacker.Hijack()
	if err != nil {
		return nil, fmt.Errorf("hijacker错误: %s", err)
	}

	return conn, nil
}

// CopyHeader 浅拷贝Header
func CopyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// CloneHeader 深拷贝Header
func CloneHeader(h http.Header) http.Header { // 可在这里修改Header
	h2 := make(http.Header, len(h))
	for k, vv := range h {
		vv2 := make([]string, len(vv))
		copy(vv2, vv)
		h2[k] = vv2
	}
	return h2
}

// CloneBody 拷贝Body
func CloneBody(b io.ReadCloser) (r io.ReadCloser, body []byte, err error) {
	if b == nil {
		return http.NoBody, nil, nil
	}
	body, err = ioutil.ReadAll(b)
	if err != nil {
		return http.NoBody, nil, err
	}

	// log.Println(string(body))
	r = ioutil.NopCloser(bytes.NewReader(body))

	return r, body, nil
}

var hopHeaders = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

func removeConnectionHeaders(h http.Header) {
	if c := h.Get("Connection"); c != "" {
		for _, f := range strings.Split(c, ",") {
			if f = strings.TrimSpace(f); f != "" {
				h.Del(f)
			}
		}
	}
}
