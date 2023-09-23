package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"AeRO/proxy/util/message"
	"AeRO/proxy/util/pipe"

	"github.com/rs/zerolog/log"

	"golang.org/x/exp/maps"
)

type Proxy struct {
	Name       string
	Type       string
	LocalAddr  string
	RemotePort string
	Started    bool
}

func (proxy *Proxy) Connect() (net.Conn, error) {
	//addr := net.JoinHostPort(proxy.LocalIp, proxy.LocalPort)
	return net.Dial("tcp", proxy.LocalAddr)
}

func (proxy *Proxy) Start() {
	proxy.Started = true
}

type Agent struct {
	client            *Client
	ClientId          string
	conn              net.Conn
	ConnTime          time.Time
	Proxies           map[string]*Proxy
	recvChan          chan message.Message
	sendChan          chan message.Message
	HeartbeatInterval time.Duration
	LastBeatTime      time.Time
	closed            bool
	closedMutex       sync.RWMutex
	mutex             sync.RWMutex
	//pipeConnPool      chan net.Conn
}

func NewAgent(conn net.Conn, client *Client) *Agent {
	agent := &Agent{
		client:            client,
		ClientId:          client.ClientId,
		conn:              conn,
		ConnTime:          time.Now(),
		Proxies:           client.Config.Proxies,
		recvChan:          make(chan message.Message, 10),
		sendChan:          make(chan message.Message, 10),
		HeartbeatInterval: time.Second * time.Duration(client.Config.HeartbeatInterval),
	}
	return agent
}

func (agent *Agent) Run() {
	defer agent.Close()
	go agent.awaitReceive()
	go agent.awaitSend()

	for _, cfg := range agent.Proxies {
		err := agent.RegisterProxy(cfg)
		if err != nil {
			log.Warn().Msgf("register proxy error:%s", err)
		}
	}

	ticker := time.NewTicker(agent.HeartbeatInterval)
	agent.LastBeatTime = time.Now().Add(agent.HeartbeatInterval / 2)
	for {
		select {
		case <-ticker.C:
			//log.Debug("schedule heartbeat")
			if time.Since(agent.LastBeatTime) > agent.HeartbeatInterval+time.Second {
				log.Warn().Msg("heartbeat timeout")
				return
			}
			req := &message.HeartbeatRequest{
				Timestamp: time.Now().UnixMicro(),
			}
			agent.sendChan <- req
			log.Debug().Msg("send heartbeat")
		case msg, ok := <-agent.recvChan:
			if !ok {
				log.Debug().Msg("receiver channel closed")
				return
			}
			switch v := msg.(type) {
			case *message.PipeMessage:
				go agent.awaitWork()
			case *message.ProxyResponse:
				if v.Result == "ok" {
					log.Info().Msgf("register proxy[%s] at [%s]", v.Name, v.Port)
					agent.mutex.RLock()
					proxy, ok := agent.Proxies[v.Name]
					agent.mutex.RUnlock()
					if ok {
						proxy.Start()
					} else {
						log.Warn().Msgf("proxy not found:%s", v.Name)
					}
				} else {
					log.Warn().Msgf("register proxy[%s] fail:%s", v.Name, v.Result)
				}
			case *message.HeartbeatRequest:
				ticker.Reset(agent.HeartbeatInterval)
				agent.LastBeatTime = time.Now()
				timestamp := agent.LastBeatTime.UnixMicro()
				log.Debug().Msgf("%s receive heartbeat latency %d", agent, timestamp-v.Timestamp)
				resp := &message.HeartbeatResponse{
					TimeSend: v.Timestamp,
					TimeRecv: timestamp,
				}
				agent.sendChan <- resp
				log.Debug().Msg("echo heartbeat")
			case *message.HeartbeatResponse:
				ticker.Reset(agent.HeartbeatInterval)
				agent.LastBeatTime = time.Now()
				timestamp := agent.LastBeatTime.UnixMicro()
				log.Debug().Msgf("heartbeat response latency %d=%d+%d", timestamp-v.TimeSend, v.TimeRecv-v.TimeSend, timestamp-v.TimeRecv)
			default:
				log.Warn().Msg("unknown message")
			}
		}
	}

}

// 连接overhead： 先导workmessage，再连接
func (agent *Agent) awaitWork() {
	conn, err := agent.client.NewPipeConn()
	if err != nil {
		log.Error().Msgf("create new pipe fail:%s", err)
		return
	}
	defer conn.Close()
	msg, err := message.Get(conn)
	if err != nil {
		log.Debug().Msgf("work closed %s", err)
		return
	}
	req, ok := msg.(*message.WorkMessage)
	if !ok {
		return
	}
	log.Info().Msgf("new work:%s", req)
	//find proxy
	agent.mutex.RLock()
	proxy, ok := agent.Proxies[req.ProxyName]
	agent.mutex.RUnlock()
	if !ok {
		log.Error().Msgf("proxy not found:%s", req.ProxyName)
		return
	}
	if !proxy.Started {
		log.Error().Msgf("proxy not started:%s", proxy.Name)
		return
	}
	dstConn, err := proxy.Connect()
	if err != nil {
		log.Error().Msgf("proxy %s@%s connect fail:%s", req.ProxyName, proxy.LocalAddr, err)
		return
	}
	pipe.Join(conn, dstConn)
	log.Info().Msgf("work done:%s", req)
}

func (agent *Agent) RegisterProxy(proxy *Proxy) error {
	req := &message.ProxyRequest{
		Name:   proxy.Name,
		Type:   proxy.Type,
		Port:   proxy.RemotePort,
		Target: proxy.LocalAddr,
		Enable: true,
	}
	err := message.Send(req, agent.conn)
	if err != nil {
		return err
	}
	//
	agent.mutex.Lock()
	agent.Proxies[proxy.Name] = proxy
	agent.mutex.Unlock()
	log.Info().Msgf("register proxy[%s@%s] type [%s,%s]", proxy.Name, proxy.LocalAddr, proxy.Type, proxy.RemotePort)
	return nil
}

func (agent *Agent) awaitReceive() {
	defer func() {
		if err := recover(); err != nil {
			log.Error().Caller().Msgf("awaitReceive panic:%s", err)
		}
	}()
	if agent.IsClosed() {
		return
	}
	for {
		if msg, err := message.Get(agent.conn); err != nil {
			if err == io.EOF {
				log.Debug().Msgf("agent connect closed")
				agent.Close()
				return
			} else {
				log.Warn().Msgf("get message error:%s", err)
			}
			break
		} else {
			if agent.IsClosed() {
				log.Debug().Msgf(fmt.Sprint("agent recved but closed:", msg))
				break
			}
			agent.recvChan <- msg
		}
	}
}

func (agent *Agent) awaitSend() {
	for {
		msg, ok := <-agent.sendChan
		if !ok { //agent closed
			break
		}
		err := message.Send(msg, agent.conn)
		if err != nil {
			log.Error().Msgf("send message error:%s", err)
			break
		}
	}
}

func (agent *Agent) IsClosed() bool {
	agent.closedMutex.RLock()
	defer agent.closedMutex.RUnlock()
	return agent.closed
}

func (agent *Agent) Close() {
	if agent.IsClosed() {
		return
	}
	agent.closedMutex.Lock()
	agent.closed = true
	agent.closedMutex.Unlock()
	close(agent.recvChan)
	close(agent.sendChan)
	agent.conn.Close()

	log.Info().Msgf("%s close", agent)
}

func (agent *Agent) String() string {
	return fmt.Sprintf("<%s:%v>", agent.ClientId, maps.Values(agent.Proxies))
}
