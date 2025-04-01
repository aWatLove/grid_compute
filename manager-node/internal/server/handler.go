package server

import (
	"encoding/json"
	"errors"
	"github.com/valyala/fasthttp"
	manager_client "manager-node/internal/manager-client"
	"manager-node/pkg/model"
	"net/http"
)

// regNodeMaster - регистрация мастер ноды
func (s *Server) regNodeMaster(method string, body []byte, args *fasthttp.Args) error {
	if method != http.MethodPost {
		return errMethodNotAllowed
	}

	var req model.Node
	err := json.Unmarshal(body, &req)
	if err != nil {
		return err
	}

	return s.managerCli.RegisterMaster(req)
}

// regNodeSlave - регистрация слейв ноды
func (s *Server) regNodeSlave(method string, body []byte, args *fasthttp.Args) error {
	if method != http.MethodPost {
		return errMethodNotAllowed
	}

	var req model.Node
	err := json.Unmarshal(body, &req)
	if err != nil {
		return err
	}

	return s.managerCli.RegisterSlave(req)
}

// removeNode - удаление ноды
func (s *Server) removeNode(method string, body []byte, args *fasthttp.Args) error { //todo
	if method != http.MethodDelete {
		return errMethodNotAllowed
	}

	//uuid := string(args.Peek("uuid"))

	return nil
}

// addTask - добавление задачи
func (s *Server) addTask(method string, body []byte, args *fasthttp.Args) error {
	if method != http.MethodPost {
		return errMethodNotAllowed
	}
	var req manager_client.TaskConfig
	err := json.Unmarshal(body, &req)
	if err != nil {
		return err
	}

	return s.managerCli.SetTask(req)
}

// closeTask - закрытие задачи от мастер ноды
func (s *Server) closeTask(method string, body []byte, args *fasthttp.Args) error {
	if method != http.MethodPost {
		return errMethodNotAllowed
	}

	uuid := string(args.Peek("uuid"))
	if uuid == "" {
		return errors.New("master uuid is required")
	}

	return s.managerCli.CloseTask(uuid)
}

// completeSubTask - подтверждение от слейв ноды, о том что подзадача решена
func (s *Server) completeSubTask(method string, body []byte, args *fasthttp.Args) error {
	if method != http.MethodPost {
		return errMethodNotAllowed
	}

	var req model.CompleteSubtaskRequest
	err := json.Unmarshal(body, &req)
	if err != nil {
		return err
	}

	return s.managerCli.CompleteSubTask(req)
}

// alertSubtaskError - уведомление о том что подзадача вернула ошибку
func (s *Server) alertSubtaskError(method string, body []byte, args *fasthttp.Args) error {
	if method != http.MethodPost {
		return errMethodNotAllowed
	}

	var req ErrorSubtaskReq
	err := json.Unmarshal(body, &req)
	if err != nil {
		return err
	}

	s.managerCli.AlertSubtaskError(req.SubtaskUUID, req.SlaveUUID, req.Error)

	return nil
}

type ErrorSubtaskReq struct {
	SlaveUUID   string `json:"SlaveUUID"`
	SubtaskUUID string `json:"SubtaskUUID"`
	Error       string `json:"Error"`
}
