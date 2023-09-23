package main

import (
	"fmt"
	"net"

	"AeRO/proxy/util/message"
	"AeRO/proxy/util/pipe"

	"github.com/rs/zerolog/log"
)

type Proxy struct {
	manager  *Manager
	Name     string
	Port     string
	Remote   string
	listener net.Listener
}

func NewProxy(name string, port string, remote string, mgr *Manager) (*Proxy, error) {
	proxy := &Proxy{
		manager: mgr,
		Name:    name,
		Port:    port,
		Remote:  remote,
	}
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, err
	}
	proxy.listener = listener
	//获取实际端口
	proxy.Port = proxy.ActualPort()
	log.Info().Msgf("%s listen on:%s", proxy.Name, listener.Addr())
	return proxy, nil
}

func (proxy *Proxy) ActualPort() string {
	_, port, err := net.SplitHostPort(proxy.listener.Addr().String())
	if err != nil {
		log.Warn().Msgf("%s listener addr err", proxy.listener.Addr())
		return proxy.Port
	}
	return port
}

func (proxy *Proxy) Run() {
	for {
		conn, err := proxy.listener.Accept()
		if err != nil {
			log.Debug().Msgf("%s listener closed", proxy)
			return
		}
		go proxy.handle(conn)
	}
}

func (proxy *Proxy) Close() {
	proxy.listener.Close()
	log.Info().Msgf("%s:%s closed", proxy.manager.ClientId, proxy)
}

func (proxy *Proxy) handle(conn net.Conn) {
	//retry pipe connect if fail to send work request
	for retry := 0; retry < 3; retry++ {
		//get pipe connect from manager
		pipeConn, err := proxy.manager.PopPipeConn()
		if err != nil {
			return
		}
		//send work request
		srcIp, srcPort, _ := net.SplitHostPort(conn.RemoteAddr().String())
		dstIp, dstPort, _ := net.SplitHostPort(conn.LocalAddr().String())
		req := &message.ConnRequest{
			ProxyName: proxy.Name,
			SrcIp:     srcIp,
			SrcPort:   srcPort,
			DstIp:     dstIp,
			DstPort:   dstPort,
		}
		if err := message.Send(req, pipeConn); err != nil {
			log.Warn().Msgf("%s send work message fail:%s", proxy, err)
			pipeConn.Close()
			continue
		}
		//connect join pipe
		pipe.Join(conn, pipeConn)
		log.Debug().Msgf("work done:%s", req)
		return
	}
}

func (proxy *Proxy) String() string {
	return fmt.Sprintf("<%s:%s>", proxy.Name, proxy.Port)
}
