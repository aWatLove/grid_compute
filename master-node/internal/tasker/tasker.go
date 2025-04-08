package tasker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"master-node/internal/config"
	"master-node/pkg/model"
	"net/http"
	"os"
	"strings"
)

const (
	STATUS_SOLVING uint8 = iota
	STATUS_DONE
	STATUS_CLOSED
	STATUS_ERROR
)

var statusStr = []string{
	"solving",
	"done",
	"close",
	"error",
}

type TaskEngine interface {
	ConfirmSubtaskHandler(json.RawMessage)
	DoneTaskHandler()
	ErrorTaskHandler(err error)
}

type Tasker struct {
	Task model.TaskConfig

	chTask  chan json.RawMessage
	chError chan error
	chDone  chan struct{}

	e TaskEngine

	cfg *config.Config

	status uint8

	ctx    context.Context
	cancel context.CancelFunc
}

func NewTasker(ctx context.Context, cfg *config.Config, e TaskEngine, taskData json.RawMessage) (*Tasker, error) {
	t := &Tasker{
		cfg:     cfg,
		e:       e,
		chDone:  make(chan struct{}),
		chTask:  make(chan json.RawMessage),
		chError: make(chan error),
	}

	ctxT, cancel := context.WithCancel(ctx)

	t.ctx = ctxT
	t.cancel = cancel

	//
	fGenerate, err := os.Open(cfg.TaskScriptGeneratePath)
	if err != nil {
		return nil, err
	}
	defer fGenerate.Close()
	dataGenerate, err := io.ReadAll(fGenerate)
	if err != nil {
		return nil, err
	}

	if !strings.Contains(string(dataGenerate), fmt.Sprintf("def %s(", cfg.TaskFuncNameGenerate)) {
		return nil, fmt.Errorf("tasker Script does not contain function")
	}

	//
	fCompute, err := os.Open(cfg.TaskScriptComputePath)
	if err != nil {
		return nil, err
	}
	defer fCompute.Close()
	dataCompute, err := io.ReadAll(fCompute)
	if err != nil {
		return nil, err
	}

	if !strings.Contains(string(dataCompute), fmt.Sprintf("def %s(", cfg.TaskFuncNameCompute)) {
		return nil, fmt.Errorf("tasker Script does not contain function")
	}
	//

	t.Task = model.TaskConfig{
		MasterUUID: cfg.UUID,
		GeneratorScript: model.ScriptConfig{
			Script:   string(dataGenerate),
			FuncName: cfg.TaskFuncNameGenerate,
		},
		ComputeScript: model.ScriptConfig{
			Script:   string(dataCompute),
			FuncName: cfg.TaskFuncNameCompute,
		},
		Data: taskData,
	}

	return t, nil
}

func (t *Tasker) Start() error {
	t.worker()

	err := t.regNode()
	if err != nil {
		return err
	}

	err = t.sendTaskToManager()
	if err != nil {
		return err
	}
	t.status = STATUS_SOLVING

	return nil
}

func (t *Tasker) worker() {
	go func() {
		for {
			select {
			case task := <-t.chTask:
				t.e.ConfirmSubtaskHandler(task)
			case <-t.chDone:
				t.status = STATUS_DONE
				t.e.DoneTaskHandler()
				return
			case err := <-t.chError:
				t.status = STATUS_ERROR
				t.e.ErrorTaskHandler(err)

			case <-t.ctx.Done():
				t.status = STATUS_CLOSED
				return
			}

		}
	}()
}

func (t *Tasker) Stop() error {
	err := t.closeTaskToManager()
	t.cancel()

	return err
}

func (t *Tasker) AddSubtask(subtask json.RawMessage) {
	t.chTask <- subtask
}

func (t *Tasker) DoneTask() {
	t.chDone <- struct{}{}
}

func (t *Tasker) ErrorTask(err error) {
	t.chError <- err
}

func (t *Tasker) GetStatus() string {
	return statusStr[t.status]
}

func (t *Tasker) regNode() error {
	node := model.Node{
		UUID:        t.cfg.UUID,
		Url:         "localhost", //todo
		PublicPort:  t.cfg.PublicPort,
		PrivatePort: t.cfg.PrivatePort,
	}

	payload, err := json.Marshal(node)

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", t.cfg.ManagerURL, t.cfg.ManagerRegPath), bytes.NewReader(payload))
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

func (t *Tasker) sendTaskToManager() error {

	payload, err := json.Marshal(t.Task)

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", t.cfg.ManagerURL, t.cfg.ManagerAddPath), bytes.NewReader(payload))
	if err != nil {
		return err
	}

	req.URL.Query().Add("uuid", t.cfg.UUID)

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

func (t *Tasker) closeTaskToManager() error {

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s", t.cfg.ManagerURL, t.cfg.ManagerClosePath), nil)
	if err != nil {
		return err
	}

	req.URL.Query().Add("uuid", t.cfg.UUID)

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
