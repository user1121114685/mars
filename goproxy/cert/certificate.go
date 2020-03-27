// Package cert 证书管理
package cert

import (
	"bufio"
	"crypto/rand"
	crand "crypto/rand"
	"io/ioutil"
	"os"

	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"
)

var (
	rootCA  *x509.Certificate
	rootKey *rsa.PrivateKey
)

func init() {
	var err error
	rootCA, err = loadRootCA()
	if err != nil {
		panic(fmt.Errorf("加载根证书失败: %s", err))
	}
	rootKey, err = loadRootKey()
	if err != nil {
		panic(fmt.Errorf("加载根证书私钥失败: %s", err))
	}
}

// Certificate 证书管理
type Certificate struct {
	Cache Cache
}

// Generate 生成证书
func (c *Certificate) Generate(host string) (*tls.Config, error) {
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

	priv, err := rsa.GenerateKey(crand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	tmpl := c.template(host)
	derBytes, err := x509.CreateCertificate(crand.Reader, tmpl, rootCA, &priv.PublicKey, rootKey)
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

	// 读取私钥文件
	keyBytes, err := os.Open("./conf/private/cert.pem")

	if err != nil {
		panic("Unable to read keyPem")
	}
	Scanner := bufio.NewScanner(keyBytes)
	Result := ""
	for Scanner.Scan() {
		txts := Scanner.Text()
		Result = Result + "\n" + txts
	}

	// 把字节流转成PEM结构
	block, rest := pem.Decode([]byte(Result))
	if len(rest) > 0 {
		panic("Unable to decode keyBytes")
	}

	return x509.ParseCertificate(block.Bytes)
}

// 加载根证书私钥
func loadRootKey() (*rsa.PrivateKey, error) {

	// 读取私钥文件
	keyBytes, err := ioutil.ReadFile("./conf/private/key.pem")
	if err != nil {
		panic("Unable to read keyPem")
	}
	// 把字节流转成PEM结构
	block, rest := pem.Decode([]byte(keyBytes))
	if len(rest) > 0 {
		panic("Unable to decode keyBytes")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func GenTLS() {
	max := new(big.Int).Lsh(big.NewInt(1), 128)   //把 1 左移 128 位，返回给 big.Int
	serialNumber, _ := rand.Int(rand.Reader, max) //返回在 [0, max) 区间均匀随机分布的一个随机值
	subject := pkix.Name{                         //Name代表一个X.509识别名。只包含识别名的公共属性，额外的属性被忽略。
		Organization:       []string{"Mars Hub"},
		OrganizationalUnit: []string{"Mars"},
		CommonName:         "ShaoXia.xyz",
	}
	template := x509.Certificate{
		SerialNumber: serialNumber, // SerialNumber 是 CA 颁布的唯一序列号，在此使用一个大随机数来代表它
		Subject:      subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour * 10),                    // 生成10年的证书
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature, //KeyUsage 与 ExtKeyUsage 用来表明该证书是用来做服务器认证的
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},               // 密钥扩展用途的序列

	}
	pk, _ := rsa.GenerateKey(rand.Reader, 2048) //生成一对具有指定字位数的RSA密钥

	//CreateCertificate基于模板创建一个新的证书
	//第二个第三个参数相同，则证书是自签名的
	//返回的切片是DER编码的证书
	derBytes, _ := x509.CreateCertificate(rand.Reader, &template, &template, &pk.PublicKey, pk) //DER 格式
	certOut, _ := os.Create("./conf/private/cert.pem")
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICAET", Bytes: derBytes})
	certOut.Close()
	keyOut, _ := os.Create("./conf/private/key.pem")
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(pk)})
	keyOut.Close()

}
