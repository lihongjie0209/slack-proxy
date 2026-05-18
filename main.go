package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"slack-proxy/config"
	"slack-proxy/handler"
)

// version is set at build time via -ldflags "-X main.version=vX.Y.Z"
var version = "dev"

func main() {
	cfgPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	mux := http.NewServeMux()
	for _, route := range cfg.Routes {
		r := route // capture loop variable
		mux.HandleFunc(r.SlackPath, handler.NewSlackHandler(r, nil))
		log.Printf("registered route: %s -> %s", r.SlackPath, r.DingTalk.Webhook)
	}

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("slack-proxy %s listening on %s", version, addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
