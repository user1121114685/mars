package filterrules

import (
	"bufio"
	"mars/internal/app/config"
	"os"

	"github.com/gogf/gf/text/gregex"
)

// Whitelist 白名单切片
var Whitelist []string // 白名单
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
		if gregex.IsMatchString(`^@@`, Txts) {
			list, err := gregex.ReplaceString(`^@@`, "", Txts)
			if err != nil {
				println(err.Error())
			}
			Whitelist = append(Whitelist, list)
		}
	}
}
