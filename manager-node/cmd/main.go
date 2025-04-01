package main

import (
	"context"
	"log"
	"manager-node/internal/config"
	manager_client "manager-node/internal/manager-client"
	"manager-node/internal/server"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// ====== Config ======
	log.Println("[SERVICE] INITIALIZING CONFIG")
	cfg := config.LoadConfig()
	// ====================

	// ===== Manager =====
	log.Println("[SERVICE] INITIALIZING GENERATOR")
	manager := manager_client.NewManagerClient(cfg)

	// =====================

	// ====== Server ======
	log.Println("[SERVICE] START SERVER")
	srv := server.New(cfg, manager)
	log.Println("[SERVER] Start")
	srv.Start()
	// ====================

	<-stop
	ctxClose, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	err := srv.Stop(ctxClose)
	if err != nil {
		log.Fatalln("[SERVER][ERROR] error while stopping: ", err)
	}

}
