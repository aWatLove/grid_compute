package server

import (
	"errors"
	"github.com/valyala/fasthttp"
	"master-node/internal/utils"
	"strings"
	"unicode/utf8"
)

const (
	// Nodes
	SUBTASK_DONE = "/subtask/done"

	TASK_DONE  = "/task/done"
	TASK_ERROR = "/task/error"

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
		s.Handler(strings.TrimPrefix(path, V1), ctx)
	}

}

func (s *Server) Handler(path string, ctx *fasthttp.RequestCtx) {

	defer utils.Recovery("SERVER")

	method, body := string(ctx.Method()), ctx.Request.Body()

	var err error
	var resp []byte
	switch path {

	case TASK_DONE:
		err = s.taskDone(method, body, ctx.QueryArgs())
	case TASK_ERROR:
		err = s.taskError(method, body, ctx.QueryArgs())
	case SUBTASK_DONE:
		err = s.subtaskDone(method, body, ctx.QueryArgs())

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
