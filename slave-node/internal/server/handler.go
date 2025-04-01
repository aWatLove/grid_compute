package server

import (
	"encoding/json"
	"fmt"
	"github.com/valyala/fasthttp"
	"net/http"
	"slave-node/pkg/model"
)

func (s *Server) addTask(method string, body []byte, args *fasthttp.Args) error {
	if method != http.MethodPost {
		return errMethodNotAllowed
	}

	var req model.ComputeRequest

	if err := json.Unmarshal(body, &req); err != nil {
		return fmt.Errorf("Invalid request format: %v", err)
	}

	err := s.generator.AddTask(req)
	if err != nil {
		return fmt.Errorf("StatusInternalServerError: %v", err)
	}

	return nil
}

func (s *Server) checkStatus(method string) ([]byte, error) {
	if method != http.MethodGet {
		return nil, errMethodNotAllowed
	}

	return json.Marshal(s.generator.CheckStatus())
}
