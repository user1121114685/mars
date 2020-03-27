package shadowsocks

import (
	
	"github.com/Yee2/shadowsocks-go"
	log "github.com/sirupsen/logrus"
	"net"
)

func ShadowsocksMain() {
	tunnel, err := shadowsocks.NewTunnel("aes-256-gcm","123456")
	if err != nil {
		println(err)
	}
	listener, err := net.Listen("tcp", "0.0.0.0:8388")
	if err != nil {
		println(err)
	}
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			println(err)
			continue
		}
		go func() {
			defer conn.Close()
			surface, err := tunnel.Shadow(conn)
			if err != nil {
				println(err)
				return
			}
			err = shadowsocks.Handle(surface)
			if err != nil {
				log.Infof("Handle 错误 %s", err)
			}
		}()

	}
}