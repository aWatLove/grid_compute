package server

import (
	"errors"
	"github.com/valyala/fasthttp"
	"slave-node/internal/utils"
	"strings"
	"unicode/utf8"
)

const (
	// Generator
	ADD_TASK_PATH     = "/addTask"
	CHECK_STATUS_PATH = "/checkStatus"
	CANCEL_TASK       = "/cancel"

	V1 = "/api/v1"
)

var (
	errNotFound         = errors.New("not found")
	errMethodNotAllowed = errors.New("method not allowed")
)

func (s *Server) Router(ctx *fasthttp.RequestCtx) {
	defer utils.Recovery("ROUTER")

	path := string(ctx.Path())
	if !utf8.ValidString(path) {
		return
	}

	switch {
	case strings.Contains(path, V1):
		//metrics.IncRpsHandle(path)
		s.Handler(strings.TrimPrefix(path, V1), ctx)
	}

}

func (s *Server) Handler(path string, ctx *fasthttp.RequestCtx) {

	defer utils.Recovery("SERVER")

	method, body := string(ctx.Method()), ctx.Request.Body()

	var err error
	var resp []byte
	switch path {
	case ADD_TASK_PATH:
		err = s.addTask(method, body, ctx.QueryArgs())
	case CHECK_STATUS_PATH:
		resp, err = s.checkStatus(method)

	default:
		err = errNotFound
	}

	if err != nil {
		resp = []byte(err.Error())
	}
	setStatusCode(ctx, err)
	if resp != nil {
		ctx.Response.SetBody(resp)
	}
}
