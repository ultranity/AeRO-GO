package main

import (
	"flag"

	"AeRO/proxy/util/zlog"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// var DefaultConfig = &ServerConfig{}
var LogFileSize int64 = 1024 * 1024 //1m
var LogBackupCount = 5

func main() {
	cfg := &ServerConfig{}
	flag.StringVar(&cfg.Ip, "ip", "0.0.0.0", "server ip")
	flag.StringVar(&cfg.Port, "port", "8080", "server port")
	flag.StringVar(&cfg.AuthCode, "auth", "", "server auth code")
	flag.IntVar(&cfg.startPort, "start", 30000, "server port range start(included)")
	flag.IntVar(&cfg.endPort, "end", 40000, "server portx dsw range end(included)")
	flag.BoolVar(&cfg.forceRange, "forceRange", false, "server port")
	flag.StringVar(&cfg.Api, "api", "localhost:3000", "server control api")
	flag.StringVar(&cfg.MuxAddr, "mux", "localhost:4000", "http mux server")
	flag.StringVar(&cfg.Domain, "domain", "", "server domain (if set, mux server use sub domain like: <tag>.<name>.<domain> to access target, otherwise use <server_ip>:<port>/<tag>/<name>)")
	flag.StringVar(&cfg.LogFile, "log", "", "log file")
	flag.BoolVar(&cfg.Debug, "debug", false, "debug mode")
	flag.Parse()

	zlog.Default()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if cfg.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	if cfg.LogFile != "" {
		if file, err := zlog.NewRotatingFile(cfg.LogFile, LogFileSize, LogBackupCount); err != nil {
			log.Error().Caller().Msgf("Config log file error:%s", err)
		} else {
			defer file.Close()
			zlog.SetOutput(file)
		}
	}
	server := NewServer(cfg)
	err := server.Run()
	if err != nil {
		log.Error().Caller().Msgf("server over:%s", err)
	}
}
