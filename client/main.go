package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"AeRO/proxy/util/zlog"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var LogFileSize int64 = 1024 * 1024
var LogBackupCount = 5

func main() {
	cfg := &ClientConfig{
		Proxies: map[string]*Proxy{},
	}
	hostname, _ := os.Hostname()
	flag.StringVar(&cfg.Name, "host", hostname, "client host metadata")
	flag.StringVar(&cfg.Tag, "tag", "aero", "client name")
	flag.StringVar(&cfg.ServerAddr, "server", "0.0.0.0:8080", "server ip:port")
	flag.StringVar(&cfg.AuthCode, "auth", "", "server auth code")
	flag.StringVar(&cfg.LogFile, "log", "", "log file")
	flag.BoolVar(&cfg.Debug, "debug", false, "debug mode")
	flag.IntVar(&cfg.HeartbeatInterval, "ping", 60, "heartbeat ping interval")
	flag.IntVar(&cfg.PoolSize, "pool", 2, "pipe pool size estimation")
	target := flag.String("target", "", "target name:ip:port list")
	flag.Parse()

	zlog.Default()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if cfg.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	if *target == "" {
		log.Error().Msg("target is empty")
		return
	}
	if cfg.Name == "" {
		log.Error().Msg("name is empty")
		return
	}
	if cfg.Tag == "" {
		log.Error().Msg("tag is empty")
		return
	}
	for i, element := range strings.Split(*target, ",") {
		split := strings.Split(element, "@")

		//name, ip, port := split[0], split[1], split[2]
		name, addr, remote := "default", "", "0"
		if len(split) == 3 {
			name, addr, remote = split[0], split[1], split[2]
		} else if len(split) == 2 {
			name, addr = split[0], split[1]
		} else {
			name = fmt.Sprint(i)
			addr = split[0]
		}
		if _, ok := cfg.Proxies[name]; ok {
			log.Error().Msgf("duplicate target name:%s", name)
			return
		}
		cfg.Proxies[name] = &Proxy{
			Name:       name,
			Type:       "tcp",
			LocalAddr:  addr,
			RemotePort: remote,
		}
	}

	if cfg.LogFile != "" {
		if file, err := zlog.NewRotatingFile(cfg.LogFile, LogFileSize, LogBackupCount); err != nil {
			log.Error().Msgf("Config log file error:%s", err)
		} else {
			defer file.Close()
			zlog.SetOutput(file)
		}
	}
	//go func() {
	//	log.Err(http.ListenAndServe("localhost:6060", nil))
	//}()
	client := NewClient(cfg)
	err := client.Run()
	if err != nil {
		log.Error().Msgf("client over:%s", err)
	}
}
