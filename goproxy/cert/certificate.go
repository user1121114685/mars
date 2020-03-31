// Package cert 证书管理
package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"mars/internal/app/config"
	"math/big"
	"net"
	"time"
)

var (
	RootCA  *x509.Certificate
	RootKey *rsa.PrivateKey
)

// Certificate 证书管理
type Certificate struct {
	Cache Cache
}

// Generate 生成证书
func (c *Certificate) Generate(host string) (*tls.Config, error) {
	if RootCA == nil || RootKey == nil {

		var err error
		RootCA, err = loadRootCA()
		if err != nil {
			panic(fmt.Errorf("加载根证书失败: %s", err))
		}
		RootKey, err = loadRootKey()
		if err != nil {
			panic(fmt.Errorf("加载根证书私钥失败: %s", err))
		}

	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	// 先从缓存中查找证书
	if cert := c.Cache.Get(host); cert != nil {
		tlsConf := &tls.Config{
			Certificates: []tls.Certificate{*cert},
		}

		return tlsConf, nil
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	tmpl := c.template(host)
	derBytes, err := x509.CreateCertificate(rand.Reader, tmpl, RootCA, &priv.PublicKey, RootKey) // 根据主证书创建证书
	if err != nil {
		return nil, err
	}
	certBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	}
	serverCert := pem.EncodeToMemory(certBlock)

	keyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	}
	serverKey := pem.EncodeToMemory(keyBlock)

	cert, err := tls.X509KeyPair(serverCert, serverKey)
	if err != nil {
		return nil, err
	}
	tlsConf := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	// 缓存证书
	c.Cache.Set(host, &cert)

	return tlsConf, nil
}

func (c *Certificate) template(host string) *x509.Certificate {
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: host,
		},
		NotBefore:             time.Now().AddDate(-1, 0, 0),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		BasicConstraintsValid: true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageDataEncipherment,
		EmailAddresses:        []string{"qingqianludao@gmail.com"},
	}

	if ip := net.ParseIP(host); ip != nil {
		cert.IPAddresses = []net.IP{ip}
	} else {
		cert.DNSNames = []string{host}
	}

	return cert
}

// 加载根证书
func loadRootCA() (*x509.Certificate, error) {

	log.Println(config.Conf)
	keyBytes, err := ioutil.ReadFile(config.Conf.Certificate.BasePrivate)

	if err != nil {
		panic("Unable to read keyPem")
	}

	block, _ := pem.Decode(keyBytes)

	return x509.ParseCertificate(block.Bytes)
}

// 加载根证书私钥
func loadRootKey() (*rsa.PrivateKey, error) {
	keyBytes, err := ioutil.ReadFile(config.Conf.Certificate.CaPrivate)

	if err != nil {
		panic("Unable to read keyPem")
	}

	block, _ := pem.Decode(keyBytes)

	return x509.ParsePKCS1PrivateKey(block.Bytes)
}
