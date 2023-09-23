package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"AeRO/proxy/util"
	"AeRO/proxy/util/message"

	"github.com/rs/zerolog/log"
	"golang.org/x/exp/maps"
)

var ErrMgrClosed = errors.New("manager is closed")
var ErrHBTimeout = errors.New("heartbeat timeout")
var ErrRecvChClosed = errors.New("receiver channel closed")
var ErrSendChClosed = errors.New("send channel closed")

var MaxPoolSize int = 8 //pipe缓冲池数量

// 一个 Manager 管理一个客户端的所有pipe和proxy;
type Manager struct {
	ClientId     string
	Tag          string
	conn         net.Conn  //control message connect
	ConnTime     time.Time //首次注册时间
	recvChan     chan message.Message
	sendChan     chan message.Message
	Proxies      map[string]*Proxy
	ProxiesStr   string
	mutex        sync.RWMutex
	pipeConnPool chan net.Conn
	PoolSize     int
	Meta         *message.LoginRequest
	//heartbeatTimer    time.Ticker //pipe的可靠性由proxy确认
	//heartbeatInterval time.Duration
	LastBeatTime time.Time
	Latency      util.EMA
	closed       bool
	closedMutex  sync.RWMutex
	done         chan *Manager //notice server close manager
}

func NewManager(id string, conn net.Conn, done chan *Manager, req *message.LoginRequest) *Manager {
	manager := &Manager{
		ClientId: id,
		Tag:      req.Tag,
		conn:     conn,
		ConnTime: time.Now(),
		PoolSize: 2,
		Meta:     req,
		done:     done,
		Proxies:  make(map[string]*Proxy),
		recvChan: make(chan message.Message, 10),
		sendChan: make(chan message.Message, 10),
	}
	if req.PoolSize > 0 {
		manager.PoolSize = req.PoolSize
	}
	if manager.PoolSize > MaxPoolSize {
		manager.PoolSize = MaxPoolSize
	}
	manager.pipeConnPool = make(chan net.Conn, manager.PoolSize*2)
	return manager
}

func (manager *Manager) Run() {
	defer func() {
		manager.done <- manager //notice server close manager
	}()
	go manager.awaitReceive()
	go manager.awaitSend()

	for i := 0; i < manager.PoolSize; i++ {
		manager.ReqPipeConn()
	}

	for {
		msg, err := manager.getMessage()
		if err != nil {
			return
		}
		var resp message.Message

		switch v := msg.(type) {
		case *message.HeartbeatRequest:
			manager.LastBeatTime = time.Now()
			timestamp := manager.LastBeatTime.UnixMicro()
			log.Debug().Msgf("%s receive heartbeat %d", manager, timestamp-v.Timestamp)
			manager.Latency.Update(timestamp - v.Timestamp)
			resp = &message.HeartbeatResponse{
				TimeSend: v.Timestamp,
				TimeRecv: timestamp,
			}
		case *message.HeartbeatResponse:
			manager.LastBeatTime = time.Now()
			timestamp := manager.LastBeatTime.UnixMicro()
			log.Debug().Msgf("heartbeat response %d=%d+%d", timestamp-v.TimeSend, v.TimeRecv-v.TimeSend, timestamp-v.TimeRecv)
			manager.Latency.Update((timestamp - v.TimeSend) / 2)
		case *message.ProxyRequest:
			if v.Enable { // new proxy
				resp = manager.RegisterProxy(v)
			} else { // close proxy
				manager.RemoveProxy(v)
			}
			manager.ListProxies()
		}

		if resp != nil {
			if err := manager.SendMassage(resp); err != nil {
				return
			}
		}
	}
}

func (manager *Manager) RegisterProxy(req *message.ProxyRequest) *message.ProxyResponse {
	proxy, err := NewProxy(req.Name, req.Port, req.Target, manager)
	if err != nil {
		resp := &message.ProxyResponse{
			Result: fmt.Sprintf("register proxy error:%s", err),
		}
		return resp
	}

	go proxy.Run()

	manager.mutex.Lock()
	manager.Proxies[proxy.Name] = proxy
	manager.mutex.Unlock()
	log.Info().Msgf("%s register proxy:%s", manager, proxy)

	return &message.ProxyResponse{
		Name:   proxy.Name,
		Port:   proxy.Port,
		Result: "ok",
	}
}

func (manager *Manager) RemoveProxy(req *message.ProxyRequest) {
	name := req.Name
	manager.mutex.Lock()
	proxy, ok := manager.Proxies[name]
	if ok {
		delete(manager.Proxies, name)
	}
	manager.mutex.Unlock()
	if !ok {
		return
	}
	proxy.Close()
	log.Info().Msgf("%s unregister proxy:%s", manager, proxy)
}

func (manager *Manager) ReqPipeConn() error {
	req := &message.PipeMessage{}
	err := manager.SendMassage(req)
	if err != nil {
		log.Error().Msgf("%s failed to send pipe req", manager.ClientId)
	}
	return err
}

func (manager *Manager) PushPipeConn(conn net.Conn) error {
	defer func() {
		if err := recover(); err != nil {
			log.Error().Caller().Msgf("PushPipeConn panic:%s", err)
		}
	}()
	if manager.IsClosed() {
		return ErrMgrClosed
	}
	select {
	case manager.pipeConnPool <- conn:
		log.Debug().Msgf("%s push a new pipe to pool", manager)
	default:
		log.Warn().Msgf("%s pipe pool is full", manager)
		return errors.New("pipe pool is full")
	}
	return nil
}

func (manager *Manager) PopPipeConn() (conn net.Conn, err error) {
	var ok bool
	// send pipe request to client to create new pipe
	manager.ReqPipeConn()
	select {
	case conn, ok = <-manager.pipeConnPool:
		if !ok {
			log.Debug().Msgf("%s pop pipe err", manager)
			err = ErrMgrClosed
			return
		}
		log.Debug().Msgf("%s pop a pipe", manager)
	case <-time.After(time.Second * 5):
		err = errors.New("await get new pipe timeout")
		log.Warn().Msgf("%s await get new pipe timeout", manager.ClientId)
	}
	return
}

func (manager *Manager) awaitReceive() {
	defer func() {
		if err := recover(); err != nil {
			log.Error().Caller().Msgf("awaitReceive panic:%s", err)
		}
	}()
	if manager.IsClosed() {
		return
	}
	for {
		if res, err := message.Get(manager.conn); err != nil {
			if err == io.EOF {
				log.Debug().Msgf("%s connect closed", manager)
			} else {
				log.Warn().Msgf("%s get message error:%s", manager, err)
			}
			manager.Close()
			return
		} else {
			if manager.IsClosed() {
				return
			}
			manager.recvChan <- res
		}
	}
}

func (manager *Manager) awaitSend() {
	for {
		resp, ok := <-manager.sendChan
		if !ok {
			log.Debug().Msgf("send channel close")
			break
		}
		err := message.Send(resp, manager.conn)
		if err != nil {
			log.Warn().Msgf("%s send message error:%s", manager, err)
			break
		}
	}
}

func (manager *Manager) getMessage() (msg message.Message, err error) {
	msg, ok := <-manager.recvChan
	if !ok {
		err = ErrRecvChClosed
	}
	// heartbeatInterval检验不必要，因为pipe的可靠性由proxy确认，同时有延迟不稳定问题
	// 	case <-manager.heartbeatTimer.C:
	// 		if time.Since(manager.lastBeatTime) > manager.heartbeatInterval+time.Second {
	// 			log.Info().Msgf("%s wait receiver heartbeat timeout", manager)
	// 			err = ErrHBTimeout
	// 		}
	return
}

func (manager *Manager) Ping() int64 {
	timestamp := time.Now().UnixMicro()
	req := &message.HeartbeatRequest{
		Timestamp: timestamp,
	}
	if err := manager.SendMassage(req); err != nil {
		return timestamp
	}
	return timestamp
}

func (manager *Manager) SendMassage(msg message.Message) error {
	defer func() {
		if err := recover(); err != nil {
			log.Error().Caller().Msgf("SendMassage panic:%s", err)
		}
	}()
	if manager.IsClosed() {
		return ErrMgrClosed
	}
	manager.sendChan <- msg
	return nil
}

func (manager *Manager) IsClosed() bool {
	manager.closedMutex.RLock()
	defer manager.closedMutex.RUnlock()
	return manager.closed
}

func (manager *Manager) Close() {
	if manager.IsClosed() {
		return
	}
	manager.closedMutex.Lock()
	manager.closed = true
	manager.closedMutex.Unlock()

	manager.conn.Close()

	close(manager.sendChan)
	close(manager.recvChan)
	close(manager.pipeConnPool)

	manager.mutex.Lock()
	for k, proxy := range manager.Proxies {
		proxy.Close()
		delete(manager.Proxies, k)
	}
	manager.mutex.Unlock()

	log.Debug().Msgf("close %s", manager)
}

func (manager *Manager) ListProxies() {
	str := make([]string, len(manager.Proxies))
	for _, proxy := range manager.Proxies {
		str = append(str, fmt.Sprintf("%s:%s", proxy.Name, proxy.Port))
	}
	manager.ProxiesStr = util.JoinX(str, ",")
}

func (manager *Manager) Info() map[string]interface{} {
	return map[string]interface{}{
		"cid":     manager.ClientId,
		"psize":   manager.PoolSize,
		"cpool":   len(manager.pipeConnPool),
		"proxies": maps.Values(manager.Proxies),
		"avglat":  manager.Latency.Avg,
		"lastlat": manager.Latency.Last,
		"conn_t":  manager.ConnTime.UnixMicro(),
		"last_t":  manager.LastBeatTime.UnixMicro(),
		"meta":    manager.Meta,
	}
}

func (manager *Manager) String() string {
	return fmt.Sprintf("<%s:%s:%s>", manager.ClientId, manager.Tag, manager.ProxiesStr)
}
