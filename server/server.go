package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"AeRO/proxy/util"
	"AeRO/proxy/util/message"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/libp2p/go-yamux/v4"

	"github.com/rs/zerolog/log"
	"golang.org/x/exp/slices"
)

type ServerConfig struct {
	Ip       string
	Port     string
	AuthCode string
	LogFile  string
	Debug    bool

	Api     string
	MuxAddr string
	Domain  string
}

type Server struct {
	config   *ServerConfig
	address  string
	listener net.Listener
	managers map[string]*Manager
	mutex    sync.RWMutex
	mgrDone  chan *Manager
}

func NewServer(config *ServerConfig) *Server {
	srv := &Server{
		config:   config,
		address:  net.JoinHostPort(config.Ip, config.Port),
		managers: make(map[string]*Manager),
		mgrDone:  make(chan *Manager, 1),
	}
	go srv.ApiServer(config.Api)
	go srv.MuxServer(config.MuxAddr, config.Domain)
	return srv
}

func (server *Server) Run() error {
	listener, err := net.Listen("tcp", server.address)
	if err != nil {
		return err
	}
	server.listener = listener
	defer listener.Close()

	log.Info().Msgf("server listen on %s", server.address)
	go server.awaitManagerDone()

	for {
		conn, err := server.listener.Accept()
		if err != nil {
			log.Warn().Msgf("listener close:%s", err)
			return err
		}
		go server.Handle(conn)
	}
}

func (server *Server) Handle(conn net.Conn) {
	cfg := yamux.DefaultConfig()
	cfg.LogOutput = io.Discard
	session, err := yamux.Server(conn, cfg, nil)
	if err != nil {
		log.Warn().Msgf("Create yamux session fail:%s", err)
		return
	}
	for {
		stream, err := session.AcceptStream()
		if err != nil {
			log.Debug().Msgf("yamux session close:%v", err)
			return
		}
		log.Debug().Msgf("Accept new stream:%d", stream.StreamID())
		go server.HandleStream(stream)
	}
}

func (server *Server) HandleStream(stream *yamux.Stream) {
	_ = stream.SetReadDeadline(time.Now().Add(time.Second * 5))
	msg, err := message.Get(stream)
	if err != nil {
		log.Error().Msgf("Get message from new stream fail:%s", err)
		stream.Close()
		return
	}
	_ = stream.SetReadDeadline(time.Time{})

	switch v := msg.(type) {
	case *message.LoginRequest:
		err := server.CheckLogin(v)
		if err != nil {
			resp := &message.LoginResponse{
				Result: fmt.Sprintf("login fail:%s", err),
			}
			message.Send(resp, stream)
			stream.Close()
			return
		}
		log.Info().Msgf("New client login; address:%s version:%d auth code:%s Os:%s", stream.RemoteAddr(), v.Version, v.AuthCode, v.OS)
		mgr := server.RegisterManager(stream, v)
		resp := &message.LoginResponse{
			ClientId: mgr.ClientId,
			Result:   "ok",
		}
		message.Send(resp, stream)
	case *message.PipeRequest:
		server.mutex.RLock()
		mgr, ok := server.managers[v.ClientId]
		server.mutex.RUnlock()
		if !ok {
			stream.Close()
		} else if err = mgr.PushPipeConn(stream); err != nil {
			stream.Close()
		}
	default:
		log.Warn().Msgf("Unknown message from new stream:%d", stream.StreamID())
		stream.Close()
	}
}

func (server *Server) CheckLogin(req *message.LoginRequest) error {
	if server.config.AuthCode != req.AuthCode {
		return errors.New("auth code error")
	}
	if !util.CheckLegal(req.Tag) {
		return errors.New("tag not match /a-zA-Z0-9_-/")
	}
	//todo: compare version

	return nil
}

func (server *Server) RegisterManager(conn net.Conn, req *message.LoginRequest) *Manager {
	manager := NewManager(server.genId(), conn, server.mgrDone, req)

	go manager.Run()

	server.mutex.Lock()
	server.managers[manager.ClientId] = manager
	server.mutex.Unlock()

	return manager
}

func (server *Server) genId() (key string) {
	for {
		key = util.RandStr8Base62()
		server.mutex.RLock()
		_, ok := server.managers[key]
		server.mutex.RUnlock()
		if !ok {
			return
		}
	}
}

func (server *Server) awaitManagerDone() {
	for {
		mgr := <-server.mgrDone
		server.mutex.Lock()
		delete(server.managers, mgr.ClientId)
		server.mutex.Unlock()
		mgr.Close()
		log.Info().Msgf("%s is done", mgr)
	}
}

func (server *Server) filterManagers(tags []string) (result []*Manager) {
	for _, mgr := range server.managers {
		if len(tags) == 0 || slices.Contains(tags, mgr.Tag) {
			result = append(result, mgr)
		}
	}
	return
}

func (server *Server) Info(tags []string) (info []interface{}) {
	for _, mgr := range server.filterManagers(tags) {
		info = append(info, mgr.Info())
	}
	return
	//return maps.Values(server.managers)
}

func (server *Server) ApiServer(addr string) {
	if addr == "" {
		return
	}

	app := fiber.New()
	app.Use(compress.New())
	//app.Use(pprof.New())
	app.Get("/metrics", monitor.New())

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	app.Get("/list", func(c *fiber.Ctx) error {
		//types := c.Query("types", "")
		//for i, element := range strings.Split(types, ",") {
		//}
		tags := c.Query("tags", "")
		if tags != "" {
			return c.JSON(fiber.Map{
				"mgrs": server.Info(strings.Split(tags, ",")),
			})
		}
		return c.JSON(fiber.Map{
			"mgrs": server.Info([]string{}),
		})
	})

	app.Get("/ping", func(c *fiber.Ctx) error {
		//types := c.Query("types", "")
		//for i, element := range strings.Split(types, ",") {
		//}
		cid := c.Query("cid", "")
		if mgr, ok := server.managers[cid]; ok {
			return c.JSON(fiber.Map{
				"pingtime": mgr.Ping(),
			})
		}
		return nil
	})
	app.Listen(addr)
	log.Info().Msgf("api listen on %s", addr)
}
