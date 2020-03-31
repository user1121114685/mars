// Package config 配置
package config

import (
	"net"
	"strings"

	"strconv"

	"mars/goutil"

	"path/filepath"

	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"
)

const (
	// 环境变量前缀
	serverEnvPrefix = "MARS"
	// 环境变量key分隔符
	serverConfigKeySeparator = "_"
)

func init() {

	viper.AutomaticEnv()
	viper.SetEnvPrefix(serverEnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", serverConfigKeySeparator))
	viper.SetConfigType("toml")
}

// RuntimeMode 运行模式
type RuntimeMode string

func (m RuntimeMode) IsDev() bool {
	return m == "dev"
}

func (m RuntimeMode) IsProd() bool {
	return m == "prod"
}

// Config 配置
type Config struct {
	// App 应用配置
	App appConfig `mapstructure:"app"`
	// Proxy 代理配置
	MITMProxy mitmProxyConfig `mapstructure:"mitmProxy"`
	// 证书配置
	Certificate CertificateConfig `mapstructure:"Certificate"`
}

type appConfig struct {
	Env           RuntimeMode
	Host          string `mapstructure:"host"`
	ProxyPort     int    `mapstructure:"proxyPort"`
	InspectorPort int    `mapstructure:"inspectorPort"`
}

type mitmProxyConfig struct {
	Enabled          bool   `mapstructure:"enabled"`
	DecryptHTTPS     bool   `mapstructure:"decryptHTTPS"`
	CertCacheSize    int    `mapstructure:"certCacheSize"`
	LeveldbDir       string `mapstructure:"leveldbDir"`
	LeveldbCacheSize int    `mapstructure:"leveldbCacheSize"`
}

// CertificateConfig 证书路径
type CertificateConfig struct {
	BasePrivate     string `mapstructure:"basePrivate"`
	CaPrivate       string `mapstructure:"caPrivate"`
	UserCertificate string `mapstructure:"userCertificate"`
}

// ProxyAddr 代理监听地址
func (ac appConfig) ProxyAddr() string {
	return net.JoinHostPort(ac.Host, strconv.Itoa(ac.ProxyPort))
}

// InspectorAddr 审查监听地址
func (ac appConfig) InspectorAddr() string {
	return net.JoinHostPort(ac.Host, strconv.Itoa(ac.InspectorPort))
}

// Conf  创建Conf变量来存放配置文件
var Conf *Config

// CreateConfig 创建配置文件
func CreateConfig(configFile string, env string) error { // 个人理解，这是一个函数，返回值是*config.Config
	currentDir, err := goutil.WorkDir()
	if err != nil {
		return err
	}
	configFile = viper.GetString("configFile")
	if !filepath.IsAbs(configFile) {
		configFile = filepath.Join(currentDir, configFile)
	}
	viper.SetConfigFile(configFile)
	log.Debugf("环境变量前缀: %s", serverEnvPrefix)
	log.Debugf("环境变量key分隔符: %s", serverConfigKeySeparator)
	log.Debugf("配置文件: %s", configFile)
	err = viper.ReadInConfig()
	if err != nil {
		return err
	}
	Conf = new(Config)
	err = viper.Unmarshal(Conf)
	if err != nil {
		return err
	}
	Conf.App.Env = RuntimeMode(viper.GetString("env"))
	return nil
}
