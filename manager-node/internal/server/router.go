package server

import (
	"errors"
	"github.com/valyala/fasthttp"
	"manager-node/internal/utils"
	"strings"
	"unicode/utf8"
)

const (
	// Nodes
	REGISTER_NODE_MASTER_PATH = "/node/register/master"
	REGISTER_NODE_SLAVE_PATH  = "/node/register/slave"
	REMOVE_NODE_PATH          = "/node/remove"
	ADD_TASK_PATH             = "/task/add"
	CLOSE_TASK_PATH           = "/task/close"
	COMPLETE_SUBTASK_PATH     = "/subtask/complete"
	ALERT_ERROR_SUBTASK_PATH  = "/subtask/error"

	CHECK_TASK_STATUS = "/task/status"

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
	case REGISTER_NODE_MASTER_PATH:
		err = s.regNodeMaster(method, body, ctx.QueryArgs())
	case REGISTER_NODE_SLAVE_PATH:
		err = s.regNodeSlave(method, body, ctx.QueryArgs())
	case REMOVE_NODE_PATH:
		err = s.removeNode(method, body, ctx.QueryArgs())
	case ADD_TASK_PATH:
		err = s.addTask(method, body, ctx.QueryArgs())
	case CLOSE_TASK_PATH:
		err = s.closeTask(method, body, ctx.QueryArgs())
	case COMPLETE_SUBTASK_PATH:
		err = s.completeSubTask(method, body, ctx.QueryArgs())
	case ALERT_ERROR_SUBTASK_PATH:
		err = s.alertSubtaskError(method, body, ctx.QueryArgs())

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
