package main

import (
	"log"

	"control_page/cmd/server"
	"control_page/config"
)

func main() {
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if err := server.Run(cfg); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
