// Package recorder 记录http transaction
package recorder

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gogf/gf/text/gregex"
	uuid "github.com/satori/go.uuid"

	"mars/goproxy"
)

const (
	contentTypeBinary = "application/octet-stream"
	contentTypePlain  = "text/plain"
)

var textMimeTypes = []string{
	"text/xml", "text/html", "text/css", "text/plain", "text/javascript",
	"application/xml", "application/json", "application/javascript", "application/x-www-form-urlencoded",
	"application/x-javascript",
}

// Transaction HTTP事务
type Transaction struct {
	// Id 唯一id
	Id string `json:"id"`
	// Req 请求
	Req *Request `json:"request"`
	// Resp 响应
	Resp *Response `json:"response"`
	// ClientIP 客户端IP
	ClientIP string `json:"client_ip"`
	// ServerIP 服务端IP
	ServerIP string `json:"server_ip"`
	// StartTime 开始时间
	StartTime time.Time `json:"start_time"`
	// Duration 持续时间
	Duration time.Duration `json:"duration"`
}

// NewTransaction 创建HTTP事务
func NewTransaction() *Transaction {
	tx := &Transaction{
		Id:   uuid.NewV4().String(),
		Req:  NewRequest(),
		Resp: NewResponse(),
	}

	return tx
}

// DumpRequest 提取request
func (tx *Transaction) DumpRequest(req *http.Request) {
	// 设置Accept-Encoding后, http.transport不会自动解压, 需要自己处理
	// 强制使用gzip
	if req.Header.Get("Accept-Encoding") != "" {
		req.Header.Set("Accept-Encoding", "gzip")
	}

	tx.Req.Method = req.Method
	tx.Req.Header = goproxy.CloneHeader(req.Header)
	tx.Req.Proto = req.Proto
	tx.Req.URL = req.URL.String()
	tx.Req.Scheme = req.URL.Scheme
	tx.Req.Host = req.URL.Host
	tx.Req.Path = req.URL.Path
	tx.Req.QueryParam = req.URL.RawQuery

	var err error
	var body []byte
	req.Body, body, err = goproxy.CloneBody(req.Body)
	contentType := getContentType(req.Header)
	tx.Req.Body.setContent(contentType, body)
	if err != nil {
		body = []byte(fmt.Sprintf("复制request body错误: %s", err))
		tx.Req.Body.setContent(contentTypePlain, body)
	}
}

// DumpRequest 提取response
func (tx *Transaction) DumpResponse(resp *http.Response, e error) {
	if e != nil {
		tx.Resp.Err = e.Error()
		return
	}
	tx.Resp.Proto = resp.Proto
	tx.Resp.Header = goproxy.CloneHeader(resp.Header)
	tx.Resp.Status = resp.Status
	tx.Resp.StatusCode = resp.StatusCode

	contentType := getContentType(resp.Header)
	if !shouldReadBody(contentType) {
		tx.Resp.Body.setContent(contentTypeBinary, nil)
		return
	}

	var err error
	var body []byte
	resp.Body, body, err = goproxy.CloneBody(resp.Body)
	tx.Resp.Body.setContent(contentType, body)
	if err != nil {
		body = []byte(fmt.Sprintf("复制response body错误: %s", err))
		tx.Resp.Body.setContent(contentTypePlain, body)
		return
	}

	if !strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
		return
	}

	b, err := gzip.NewReader(bytes.NewReader(tx.Resp.Body.Content))
	if err != nil {
		body = []byte(fmt.Sprintf("解压response body错误: %s", err))
		tx.Resp.Body.setContent(contentTypePlain, body)
		return
	}
	body, err = ioutil.ReadAll(b)
	if e != nil {
		body = []byte(fmt.Sprintf("读取解压后的response body错误: %s", err))
		tx.Resp.Body.setContent(contentTypePlain, body)
		return
	}
	// body = []byte(MarsReplaceString(body)) // 参数含义 需要替换的字符  替换后的字符 目标处理文件
	// 可以从这里用正则替换body(网页实心内容)

	tx.Resp.Body.setContent(contentType, body)
}

// MarsReplaceString 正则替换结果
func MarsReplaceString(body []byte) string {
	s1, err := gregex.ReplaceString(`.*`, "我的mars成功了！！", string(body))
	if err != nil {
		log.Println(err)
	}
	return s1
}

// body是否是二进制内容
func IsBinaryBody(contentType string) bool {
	for _, item := range textMimeTypes {
		if item == contentType {
			return false
		}
	}

	return true
}

// 是否应该读取Body内容
func shouldReadBody(contentType string) bool {
	return strings.HasPrefix(contentType, "image/") || !IsBinaryBody(contentType)
}

// 获取body类型
func getContentType(h http.Header) string {
	ct := h.Get("Content-Type")
	segments := strings.Split(strings.TrimSpace(ct), ";")
	if len(segments) > 0 && segments[0] != "" {
		return strings.TrimSpace(segments[0])
	}

	return contentTypeBinary
}
