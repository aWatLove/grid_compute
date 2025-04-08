package main

import (
	"context"
	"encoding/json"
	"log"
	"master-node/internal/config"
	"master-node/internal/server"
	"master-node/internal/tasker"
	tasker_impl "master-node/internal/tasker-impl"
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

	// === tasker impl ===

	taskImpl := tasker_impl.New(cfg)

	// === === === === ===

	//matrix := map[string][][]int{
	//	"matrix": {
	//		{0, 1, 2, 3, 2, 1},
	//		{1, 0, 3, 1, 2, 2},
	//		{2, 3, 0, 2, 3, 2},
	//		{3, 1, 2, 0, 2, 2},
	//		{2, 2, 3, 2, 0, 2},
	//		{1, 2, 3, 2, 2, 0},
	//	},
	//}

	//matrix := map[string][][]int{
	//	"matrix": {
	//		{0, 1, 2, 3, 2},
	//		{1, 0, 3, 1, 2},
	//		{2, 3, 0, 2, 3},
	//		{3, 1, 2, 0, 2},
	//		{2, 2, 3, 2, 0},
	//	},
	//}

	//matrix := map[string][][]int{
	//	"matrix": {
	//		{0, 1, 2, 3},
	//		{1, 0, 3, 1},
	//		{2, 3, 0, 2},
	//		{3, 1, 2, 0},
	//	},
	//}

	//matrix := map[string][][]int{
	//	"matrix": {
	//		{0, 10, 10, 1},
	//		{10, 0, 10, 1},
	//		{10, 10, 0, 1},
	//		{1, 1, 1, 0},
	//	},
	//}

	matrix := map[string][][]int{
		"matrix": {
			{0, 0, 0, 10},
			{0, 0, 10, 1},
			{0, 10, 0, 1},
			{10, 1, 1, 0},
		},
	}

	//matrix := map[string][][]int{
	//	"matrix": {
	//		{0, 1, 2},
	//		{1, 0, 3},
	//		{2, 3, 0},
	//	},
	//}
	dataTask, err := json.Marshal(matrix)

	// ===== Tasker =====
	log.Println("[SERVICE] INITIALIZING TASKER")
	//t, err := tasker.NewTasker(context.Background(), cfg, tasker.TaskEngineMock{})
	t, err := tasker.NewTasker(context.Background(), cfg, taskImpl, dataTask)
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
