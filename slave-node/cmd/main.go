package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"slave-node/internal/config"
	"slave-node/internal/generator"
	"slave-node/internal/server"
	"slave-node/pkg/model"
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

	// ===== Generator =====
	log.Println("[SERVICE] INITIALIZING GENERATOR")
	g := generator.NewGenerator()

	// =====================

	// ===== Register Node =====
	log.Println("[SERVICE] REGISTERING NODE")
	err := registerNode(cfg)
	if err != nil {
		log.Println(err)
	}
	// =====================

	// ====== Server ======
	log.Println("[SERVICE] START SERVER")
	srv := server.New(cfg, g)
	log.Println("[SERVER] Start")
	srv.Start()
	// ====================

	<-stop
	ctxClose, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	err = srv.Stop(ctxClose)
	if err != nil {
		log.Fatalln("[SERVER][ERROR] error while stopping: ", err)
	}

}

func registerNode(cfg *config.Config) error {

	node := model.Node{
		UUID:        cfg.UUID,
		Url:         "localhost", //todo
		PublicPort:  cfg.PublicPort,
		PrivatePort: cfg.PrivatePort,
	}

	payload, err := json.Marshal(node)

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", cfg.ManagerURL, cfg.ManagerRegPath), bytes.NewReader(payload))
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("request failed: %s", string(body))
	}

	return nil
}
