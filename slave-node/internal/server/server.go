package server

import (
	"context"
	"fmt"
	"github.com/valyala/fasthttp"
	"log"
	"net/http"
	"slave-node/internal/config"
	"slave-node/internal/generator"
)

type Server struct {
	HttpServer *fasthttp.Server
	Debug      *http.Server
	Cfg        *config.Config
	generator  *generator.Generator
}

type ServerPrivate struct {
	HttpServer *http.Server
}

func New(cfg *config.Config, gen *generator.Generator) *Server {
	return &Server{
		HttpServer: new(fasthttp.Server),
		Debug: &http.Server{
			Addr: cfg.PrivatePort,
		},
		Cfg:       cfg,
		generator: gen,
	}
}

func NewServerPrivate(port string) *ServerPrivate {
	return &ServerPrivate{
		HttpServer: &http.Server{
			Addr: port,
		},
	}
}

func (s *Server) Start() {
	s.HttpServer.Handler = s.Router
	s.Debug.Handler = s.initRoutsServerPrivate()

	go func() {
		if err := s.HttpServer.ListenAndServe(
			fmt.Sprintf("0.0.0.0%s", s.Cfg.PublicPort),
		); err != nil {
			log.Fatal(err)
		}
	}()

	go func() {
		if err := s.Debug.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()
}

func setStatusCode(ctx *fasthttp.RequestCtx, err error) {
	if err != nil {
		switch err {
		case errNotFound:
			ctx.SetStatusCode(fasthttp.StatusNotFound)
		case errMethodNotAllowed:
			ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		default:
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		}
	} else {
		ctx.SetStatusCode(fasthttp.StatusOK)
	}
}

func (s *Server) initRoutsServerPrivate() http.Handler {
	privateMux := http.NewServeMux()
	privateMux.HandleFunc("/health", func(writer http.ResponseWriter, request *http.Request) { writer.WriteHeader(http.StatusOK) })

	return privateMux
}

func (s *Server) Stop(ctx context.Context) error {
	err := s.Debug.Shutdown(ctx)
	if err == http.ErrServerClosed {
		log.Println("[SERVER][STOP] HTTP private server stopped")
	} else {
		return err
	}

	err = s.HttpServer.Shutdown()
	if err == http.ErrServerClosed {
		log.Println("[SERVER][STOP] HTTP public server stopped")
		return nil
	} else {
		return err
	}
}
