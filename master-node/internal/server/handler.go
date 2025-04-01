package server

import (
	"encoding/json"
	"errors"
	"github.com/valyala/fasthttp"
	"net/http"
)

func (s *Server) taskDone(method string, body []byte, args *fasthttp.Args) error {
	if method != http.MethodPost {
		return errMethodNotAllowed
	}

	s.taskCli.DoneTask()

	return nil
}

func (s *Server) taskError(method string, body []byte, args *fasthttp.Args) error {
	if method != http.MethodPost {
		return errMethodNotAllowed
	}

	var errStr string
	err := json.Unmarshal(body, &errStr)
	if err != nil {
		return err
	}

	s.taskCli.ErrorTask(errors.New(errStr))

	return nil
}

func (s *Server) subtaskDone(method string, body []byte, args *fasthttp.Args) error {
	if method != http.MethodPost {
		return errMethodNotAllowed
	}

	var reqBody json.RawMessage
	err := json.Unmarshal(body, &reqBody)
	if err != nil {
		return err
	}

	s.taskCli.AddSubtask(reqBody)

	return nil
}
