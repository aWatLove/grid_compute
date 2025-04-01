package main

import (
	"context"
	"log"
	"master-node/internal/config"
	"master-node/internal/server"
	"master-node/internal/tasker"
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

	// ===== Tasker =====
	log.Println("[SERVICE] INITIALIZING TASKER")
	t, err := tasker.NewTasker(context.Background(), cfg, tasker.TaskEngineMock{})
	if err != nil {
		log.Fatal(err)
	}
	// =====================

	err = t.Start()
	if err != nil {
		log.Fatalln(err)
	}

	// ====== Server ======
	log.Println("[SERVICE] START SERVER")
	srv := server.New(cfg, t)
	log.Println("[SERVER] Start")
	srv.Start()
	// ====================

	<-stop
	t.Stop()

	ctxClose, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	err = srv.Stop(ctxClose)
	if err != nil {
		log.Fatalln("[SERVER][ERROR] error while stopping: ", err)
	}

}
