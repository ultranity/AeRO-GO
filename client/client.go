package main

import (
	"errors"
	"io"
	"net"
	"os"
	"time"

	"AeRO/proxy/util/message"

	"github.com/hashicorp/yamux"
	"github.com/rs/zerolog/log"
)

type ClientConfig struct {
	Name              string
	Tag               string
	ServerAddr        string
	AuthCode          string
	PoolSize          int
	LogFile           string
	Proxies           map[string]*Proxy
	HeartbeatInterval int
	Debug             bool
}

type Client struct {
	Config     *ClientConfig
	ClientId   string
	ServerAddr string
	conn       net.Conn
	session    *yamux.Session
}

func NewClient(cfg *ClientConfig) *Client {
	return &Client{
		Config:     cfg,
		ServerAddr: cfg.ServerAddr,
	}
}

func (client *Client) Run() error {
	tryCount := 0

	for {
		tryCount += 1
		if tryCount > 5 {
			return errors.New("login fail for 5 times")
		}
		conn, err := client.connectServer()
		if err == nil {
			err = client.Login(conn)
			if err == nil {
				manager := NewAgent(conn, client)
				manager.Run()
				tryCount = 0
				log.Info().Msgf("%s over", manager)
				time.Sleep(time.Second)
				continue
			} else {
				log.Warn().Msgf("login fail:%s", err)
			}
		} else {
			log.Warn().Msgf("connect server fail:%s", err)
		}
		log.Info().Msgf("try again after 6 second [%d/5]", tryCount)
		time.Sleep(time.Second * 6)
	}
}

func (client *Client) connectServer() (net.Conn, error) {
	conn, err := net.Dial("tcp", client.ServerAddr)
	if err != nil {
		return nil, err
	}
	client.conn = conn

	cfg := yamux.DefaultConfig()
	cfg.LogOutput = io.Discard
	session, err := yamux.Client(conn, cfg)
	if err != nil {
		return nil, err
	}
	client.session = session
	stream, err := session.OpenStream()
	if err != nil {
		return nil, err
	}
	return stream, nil
}

func (client *Client) NewPipeConn() (net.Conn, error) {
	log.Debug().Msgf("create new pipe connect")
	stream, err := client.session.OpenStream()
	if err != nil {
		return nil, err
	}
	msg := &message.PipeRequest{
		ClientId: client.ClientId,
	}
	err = message.Send(msg, stream)
	if err != nil {
		log.Warn().Msgf("Send pipe message fail:%s", err)
		stream.Close()
		return nil, err
	}
	return stream, nil
}

func (client *Client) Login(conn net.Conn) (err error) {
	defer func() {
		if err != nil {
			conn.Close()
		}
	}()

	req := &message.LoginRequest{
		Version:  1,
		AuthCode: client.Config.AuthCode,
		PoolSize: client.Config.PoolSize,
		HostName: client.Config.Name,
		OS:       os.Getenv("GOOS"),
		Tag:      client.Config.Tag,
	}
	err = message.Send(req, conn)
	if err != nil {
		return
	}
	conn.SetReadDeadline(time.Now().Add(time.Second * 5))
	msg, err := message.Get(conn)
	if err != nil {
		log.Warn().Msgf("Get login result error:%s", err)
		return
	}
	conn.SetReadDeadline(time.Time{})
	if resp, ok := msg.(*message.LoginResponse); !ok {
		return errors.New("invalid login response")
	} else if resp.Result != "ok" {
		return errors.New(resp.Result)
	} else {
		client.ClientId = resp.ClientId
	}
	log.Info().Msgf("login success: %s", client.ClientId)
	return nil
}
