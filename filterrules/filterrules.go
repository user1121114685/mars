package filterrules

import (
	"bufio"
	"mars/internal/app/config"
	"os"
	"strings"

	"github.com/gogf/gf/text/gregex"
)

// Whitelist 白名单切片
var Whitelist []string // 白名单

// Blacklist 需要进行过滤的名单
var Blacklist []string

// Hostlist 单网 +urL 址屏蔽
var Hostlist []string // Hosts 屏蔽方法

// ReqURLRw 对网址的url进行重写
var ReqURLRw []map[string]string

// ReqURLTo 对网址的url进行重写
var ReqURLTo []map[string]string

// ReqDel 删除 Request Header
var ReqDel []map[string]string

// ReqOriSet 原始值+增加 Request Header
var ReqOriSet []map[string]string

// ReqNewSet 新的值 Request Header
var ReqNewSet []map[string]string

// RespDel 删除 Response Header
var RespDel []map[string]string

// RespOriSet 原始 + 增加 Response Header
var RespOriSet []map[string]string

// RespNewSet 新设置 Response Header
var RespNewSet []map[string]string

// RespRw 重写 Response Body
var RespRw []map[string]string

// ReqRw 重写 Response Body
var ReqRw []map[string]string

// LoadFilterRules 加载过滤规则
func LoadFilterRules() {

	file, err := os.Open(config.Conf.Filterrules.Filepath)
	if err != nil {
		println(err.Error())
	}
	defer file.Close()
	Scanner := bufio.NewScanner(file)
	for Scanner.Scan() {
		var Txts string
		Txts = Scanner.Text()
		if !gregex.IsMatchString(`^#`, Txts) { //注释符号
			// 白名单
			if gregex.IsMatchString(`^@@`, Txts) {
				list, err := gregex.ReplaceString(`^@@`, "", Txts)
				if err != nil {
					println(err.Error())
				}
				Whitelist = append(Whitelist, list)
				continue //  continue 忽略剩余的循环体而直接进入下一次循环的过程
			}
			// Host 屏蔽方式
			if gregex.IsMatchString(`^\|\|`, Txts) {
				list, err := gregex.ReplaceString(`^\|\|`, "", Txts)
				if err != nil {
					println(err.Error())
				}
				Hostlist = append(Hostlist, list)
				//  将Host 域名加入 需要封锁的列表
				list, err = gregex.ReplaceString(`/.*`, "", Txts)
				if err != nil {
					println(err.Error())
				}
				Blacklist = append(Blacklist, list)
				continue //  continue 忽略剩余的循环体而直接进入下一次循环的过程
			}
			// URL重写
			if gregex.IsMatchString(`@url\|\|rw@`, Txts) {
				list := strings.Split(Txts, "@url||rw@")
				listRW := strings.Split(list[1], "@@@") // 此处有误？
				ReqURLRw = append(ReqURLRw, map[string]string{"url": list[0], "target": listRW[0], "result": listRW[1]})
				//  将Host 域名加入 需要封锁的列表

				newlist, err := gregex.ReplaceString(`/.*`, "", list[0])
				if err != nil {
					println(err.Error())
				}
				Blacklist = append(Blacklist, newlist)
				continue //  continue 忽略剩余的循环体而直接进入下一次循环的过程
			}
			// URL重定向
			if gregex.IsMatchString(`@url\|\|to@`, Txts) {
				list := strings.Split(Txts, "@url||to@")
				listRW := strings.Split(list[1], "@@@")
				urltohost, err := gregex.ReplaceString(`/.*`, "", listRW[1]) // 将需要重定向的域名提出来
				if err != nil {
					println(err.Error())
				}
				urltopath, err := gregex.ReplaceString(`.*/+?`, "", listRW[1]) // 将重定向的path 提出来
				if err != nil {
					println(err.Error())
				}
				ReqURLTo = append(ReqURLTo, map[string]string{"url": list[0], "target": listRW[0], "result": listRW[1], "urltohost": urltohost, "urltopath": urltopath})
				//  将Host 域名加入 需要封锁的列表

				newlist, err := gregex.ReplaceString(`/.*`, "", list[0])
				if err != nil {
					println(err.Error())
				}
				Blacklist = append(Blacklist, newlist)
				continue //  continue 忽略剩余的循环体而直接进入下一次循环的过程
			}

			// Request Headers 删除
			if gregex.IsMatchString(`@req\|\|del@`, Txts) {
				list := strings.Split(Txts, "@req||del@")

				ReqDel = append(ReqDel, map[string]string{"url": list[0], "headerName": list[1]})
				//  将Host 域名加入 需要封锁的列表

				newlist, err := gregex.ReplaceString(`/.*`, "", list[0])
				if err != nil {
					println(err.Error())
				}
				Blacklist = append(Blacklist, newlist)
				continue //  continue 忽略剩余的循环体而直接进入下一次循环的过程
			}
			// Request Headers 追加设置
			if gregex.IsMatchString(`@req\|\|oriset@`, Txts) {
				list := strings.Split(Txts, "@req||oriset@")
				listRW := strings.Split(list[1], "@@@")
				ReqOriSet = append(ReqOriSet, map[string]string{"url": list[0], "target": listRW[0], "result": listRW[1]})
				//  将Host 域名加入 需要封锁的列表

				newlist, err := gregex.ReplaceString(`/.*`, "", list[0])
				if err != nil {
					println(err.Error())
				}
				Blacklist = append(Blacklist, newlist)
				continue //  continue 忽略剩余的循环体而直接进入下一次循环的过程
			}
			// Request Headers 新设置
			if gregex.IsMatchString(`@req\|\|newset@`, Txts) {
				list := strings.Split(Txts, "@req||newset@")
				listRW := strings.Split(list[1], "@@@")
				ReqNewSet = append(ReqNewSet, map[string]string{"url": list[0], "target": listRW[0], "result": listRW[1]})
				//  将Host 域名加入 需要封锁的列表

				newlist, err := gregex.ReplaceString(`/.*`, "", list[0])
				if err != nil {
					println(err.Error())
				}
				Blacklist = append(Blacklist, newlist)
				continue //  continue 忽略剩余的循环体而直接进入下一次循环的过程
			}

			// Response Headers 删除
			if gregex.IsMatchString(`@resp\|\|del@`, Txts) {
				list := strings.Split(Txts, "@resp||del@")
				RespDel = append(RespDel, map[string]string{"url": list[0], "headerName": list[1]})
				//  将Host 域名加入 需要封锁的列表

				newlist, err := gregex.ReplaceString(`/.*`, "", list[0])
				if err != nil {
					println(err.Error())
				}
				Blacklist = append(Blacklist, newlist)
				continue //  continue 忽略剩余的循环体而直接进入下一次循环的过程
			}
			// Response Headers 追加设置
			if gregex.IsMatchString(`@resp\|\|oriset@`, Txts) {
				list := strings.Split(Txts, "@resp||oriset@")
				listRW := strings.Split(list[1], "@@@")
				RespOriSet = append(RespOriSet, map[string]string{"url": list[0], "target": listRW[0], "result": listRW[1]})
				//  将Host 域名加入 需要封锁的列表

				newlist, err := gregex.ReplaceString(`/.*`, "", list[0])
				if err != nil {
					println(err.Error())
				}
				Blacklist = append(Blacklist, newlist)
				continue //  continue 忽略剩余的循环体而直接进入下一次循环的过程
			}
			// Response Headers 新设置
			if gregex.IsMatchString(`@resp\|\|newset@`, Txts) {
				list := strings.Split(Txts, "@resp||newset@")
				listRW := strings.Split(list[1], "@@@")
				RespNewSet = append(RespNewSet, map[string]string{"url": list[0], "target": listRW[0], "result": listRW[1]})
				//  将Host 域名加入 需要封锁的列表

				newlist, err := gregex.ReplaceString(`/.*`, "", list[0])
				if err != nil {
					println(err.Error())
				}
				Blacklist = append(Blacklist, newlist)
				continue //  continue 忽略剩余的循环体而直接进入下一次循环的过程
			}

			// Request Body 新设置
			if gregex.IsMatchString(`@req\|\|rw@`, Txts) {
				list := strings.Split(Txts, "@req||rw@")
				listRW := strings.Split(list[1], "@@@")
				ReqRw = append(ReqRw, map[string]string{"url": list[0], "target": listRW[0], "result": listRW[1]})
				//  将Host 域名加入 需要封锁的列表

				newlist, err := gregex.ReplaceString(`/.*`, "", list[0])
				if err != nil {
					println(err.Error())
				}
				Blacklist = append(Blacklist, newlist)
				continue //  continue 忽略剩余的循环体而直接进入下一次循环的过程
			}

			// Response Body 新设置
			if gregex.IsMatchString(`@resp\|\|rw@`, Txts) {
				list := strings.Split(Txts, "@resp||rw@")
				listRW := strings.Split(list[1], "@@@")
				RespRw = append(RespRw, map[string]string{"url": list[0], "target": listRW[0], "result": listRW[1]})
				//  将Host 域名加入 需要封锁的列表

				newlist, err := gregex.ReplaceString(`/.*`, "", list[0])
				if err != nil {
					println(err.Error())
				}
				Blacklist = append(Blacklist, newlist)
				continue //  continue 忽略剩余的循环体而直接进入下一次循环的过程
			}

		}
	}
}
