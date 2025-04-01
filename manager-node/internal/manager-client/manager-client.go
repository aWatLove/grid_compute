package manager_client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	uuid2 "github.com/google/uuid"
	"io"
	"log"
	"manager-node/internal/config"
	"manager-node/pkg/model"
	"net/http"
	"sync"
	"time"
)

const (
	STATUS_WAIT uint8 = iota
	STATUS_SOLVING
	STATUS_ERROR
	STATUS_DONE
)

var statusStr = []string{
	"wait",
	"solving",
	"error",
	"done",
}

const (
	errSubtaskThreshold = 3
	defaultSlavePower   = 50
)

type ManagerClient struct {
	MasterNodes map[string]*MasterNode
	SlaveNodes  map[string]*SlaveNode
	cfg         *config.Config

	FreeSlaves map[string]struct{} // свободные слейвы
	WorkSlaves map[string]struct{} // слейвы в работе

	taskStatus     map[string]Task    // общие задачи
	subtasksStatus map[string]Subtask // подзадачи

	mu sync.Mutex
}

func NewManagerClient(cfg *config.Config) *ManagerClient {
	mc := &ManagerClient{
		MasterNodes:    make(map[string]*MasterNode),
		SlaveNodes:     make(map[string]*SlaveNode),
		taskStatus:     make(map[string]Task),
		subtasksStatus: make(map[string]Subtask),
		FreeSlaves:     make(map[string]struct{}),
		WorkSlaves:     make(map[string]struct{}),
		cfg:            cfg,
	}

	go mc.checkMasterHealthWorker() // воркер проверки жизни Мастер-нод
	go mc.checkSlaveHealthWorker()  // воркер проверки жизни Слейв-нод

	go mc.subtaskWorker() // воркер проверки статусов выполнения подзадач

	return mc
}

type Task struct {
	MasterUuid string
	Data       json.RawMessage
	counter    uint32
}

func (t *Task) GetNext() {

}

type Subtask struct {
	uuid          string
	TaskUuid      string
	SlaveNodeUuid string
	Url           string
	start         uint32
	amount        uint32
	sendTime      time.Time
	doneTime      time.Time
	status        uint8
	errCount      int
}

type MasterNode struct {
	model.Node
	generatorScript model.ScriptConfig
	computeScript   model.ScriptConfig
	taskName        string
	statusStr       string
	status          uint8
	task            Task
	cancelTask      context.CancelFunc
}

type TaskConfig struct {
	masterUUID      string
	generatorScript model.ScriptConfig
	computeScript   model.ScriptConfig
	data            json.RawMessage
	taskName        string
	task            Task
}

type ScriptConfig struct {
	Script   string `json:"Script"`
	FuncName string `json:"FuncName"`
}

type SlaveNode struct {
	model.Node
	status string
	power  uint32
}

func (mc *ManagerClient) taskWorker(ctx context.Context, uuid string) {
	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			if task, ok := mc.taskStatus[uuid]; ok {

				for slaveUuid, _ := range mc.FreeSlaves {
					if slave, okS := mc.SlaveNodes[slaveUuid]; okS {
						mc.mu.Lock()
						amount := slave.power
						if amount == 0 {
							amount = defaultSlavePower
						}
						start := task.counter
						task.counter += amount

						delete(mc.FreeSlaves, slaveUuid)

						mc.WorkSlaves[slaveUuid] = struct{}{}
						mc.mu.Unlock()

						mc.sendSubTask(uuid2.NewString(), uuid, slave, amount, start)

					}
				}

			} else {
				log.Printf("[taskWorker][%s] task not found. Task finished\n", uuid)
				mc.alertTaskError(uuid, "task not found")
				return
			}

		}
	}

}

func (mc *ManagerClient) sendSubTask(subtaskUuid string, taskUuid string, slave *SlaveNode, amount, start uint32) {
	var subtask Subtask
	if _, ok := mc.subtasksStatus[subtaskUuid]; !ok {
		subtask = Subtask{
			uuid:          subtaskUuid,
			TaskUuid:      taskUuid,
			SlaveNodeUuid: slave.Uuid,
			Url:           slave.Url + slave.PublicPort,
			start:         start,
			amount:        amount,
			errCount:      0,
		}
		mc.subtasksStatus[subtaskUuid] = subtask
	} else {
		subtask = mc.subtasksStatus[subtaskUuid]
		subtask.Url = slave.Url + slave.PublicPort
		mc.subtasksStatus[subtaskUuid] = subtask
	}

	task := mc.taskStatus[subtask.TaskUuid]
	mn, ok := mc.MasterNodes[task.MasterUuid]
	if !ok {
		mc.alertTaskError(taskUuid, "master node not found")
	}

	reqBody := model.ComputeRequest{
		Generate: mn.generatorScript,
		Compute:  mn.computeScript,
		Data:     task.Data,
		Amount:   amount,
		Start:    start,
	}
	err := mc.sendSlave(reqBody, slave)
	if err != nil {
		log.Println(err)
		mc.AlertSubtaskError(subtaskUuid, slave.Uuid, "error send subtask to slave")
	}

}

// sendSlave - отправка подзадачи на слейв
func (mc *ManagerClient) sendSlave(reqBody model.ComputeRequest, node *SlaveNode) error {
	data, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s%s%s", node.Url, node.PublicPort, "/api/v1/addTask"), bytes.NewReader(data)) //todo checkme чекнуть как порт будет передаваться {"8080"/":8080"}
	if err != nil {
		return err
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode/100 != 2 {
		respBody, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()

		return errors.New(string(respBody))
	}

	return nil
}

// sendMasterSubTask - отправка решенного куска мастеру для дальнейшего мержа
func (mc *ManagerClient) sendMasterSubTask(data json.RawMessage, masterUuid string) error {
	master, ok := mc.MasterNodes[masterUuid]
	if !ok {
		return fmt.Errorf("master node not found")
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s%s%s", master.Url, master.PublicPort, "/api/v1/subtask/done"), bytes.NewReader(data)) //todo checkme чекнуть как порт будет передаваться {"8080"/":8080"}
	if err != nil {
		return err
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode/100 != 2 {
		respBody, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()

		return errors.New(string(respBody))
	}

	return nil
}

/*
AlertSubtaskError - Уведомление подзадачи об ошибке

Если в подзадаче ошибок больше чем 'errSubtaskThreshold', то оповещается об ошибке, вся задача

Иначе, подзадача отправляется другому слейву
*/
func (mc *ManagerClient) AlertSubtaskError(uuid string, slaveUuid string, errorStr string) {
	if _, ok := mc.SlaveNodes[slaveUuid]; ok {
		mc.mu.Lock()

		delete(mc.WorkSlaves, slaveUuid)
		mc.FreeSlaves[slaveUuid] = struct{}{}

		mc.mu.Unlock()
	} else {
		delete(mc.WorkSlaves, slaveUuid)
		return
	}

	if v, ok := mc.subtasksStatus[uuid]; ok {
		v.errCount++
		if v.errCount > errSubtaskThreshold {
			mc.alertTaskError(v.TaskUuid, errorStr)

			mc.mu.Lock()
			delete(mc.subtasksStatus, uuid)
			mc.mu.Unlock()
		} else {
			for newSlaveUuid, _ := range mc.FreeSlaves {
				mc.mu.Lock()

				slave, ok := mc.SlaveNodes[newSlaveUuid]
				if !ok {
					delete(mc.FreeSlaves, newSlaveUuid)
					mc.mu.Unlock()
					continue
				}

				delete(mc.FreeSlaves, newSlaveUuid)
				mc.WorkSlaves[newSlaveUuid] = struct{}{}
				mc.mu.Unlock()

				mc.sendSubTask(v.uuid, v.TaskUuid, slave, v.amount, v.start)
				break
			}

		}

	}

}

func (mc *ManagerClient) alertTaskError(uuid string, errorStr string) { //todo отправка уведомления об ошибке мастеру и удаление задачи
	// найти мастера по uuid , отправить ему ошибку по ручке
	//todo
}

func (mc *ManagerClient) doneTask(uuid string) { //todo отправка уведомления об окончании решения мастеру и удаление задачи
}

func (mc *ManagerClient) CompleteSubTask(resp model.CompleteSubtaskRequest) error {
	mc.mu.Lock()
	if _, ok := mc.SlaveNodes[resp.SlaveUUID]; ok {
		delete(mc.WorkSlaves, resp.SlaveUUID)
		mc.FreeSlaves[resp.SlaveUUID] = struct{}{}
	}
	mc.mu.Unlock()

	subtask, ok := mc.subtasksStatus[resp.SubtaskUUID]
	if !ok {
		return errors.New("subtask not found")
	}
	delete(mc.subtasksStatus, resp.SubtaskUUID)

	task, ok := mc.taskStatus[subtask.TaskUuid]
	if !ok {
		return errors.New("task not found")
	}

	return mc.sendMasterSubTask(resp.Data, task.MasterUuid)
}

// subtaskWorker - воркер чекинга состояния решения подзадач у слейвов
func (mc *ManagerClient) subtaskWorker() { // здесь пингуются сабтаски
	ticker := time.NewTicker(5 * time.Second)

	client := &http.Client{}

	for {
		select {
		case <-ticker.C:
			if len(mc.subtasksStatus) != 0 {
				for _, subtask := range mc.subtasksStatus {

					req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s%s", subtask.Url, "/api/v1/checkStatus"), nil)
					if err != nil {
						log.Printf("[SUBTASK_WORKER][TASK | %s][SUBTASK | %s][SLAVE | %s] error: %v\n", subtask.TaskUuid, subtask.uuid, subtask.SlaveNodeUuid, err)
						mc.AlertSubtaskError(subtask.uuid, subtask.SlaveNodeUuid, err.Error())
						continue
					}

					resp, err := client.Do(req)
					if err != nil {
						log.Printf("[SUBTASK_WORKER][TASK | %s][SUBTASK | %s][SLAVE | %s] error: %v\n", subtask.TaskUuid, subtask.uuid, subtask.SlaveNodeUuid, err)
						mc.AlertSubtaskError(subtask.uuid, subtask.SlaveNodeUuid, err.Error())
						continue
					}
					if resp.StatusCode/100 != 2 {
						respBody, _ := io.ReadAll(resp.Body)
						defer resp.Body.Close()
						log.Printf("[SUBTASK_WORKER][TASK | %s][SUBTASK | %s][SLAVE | %s] error: %v\n", subtask.TaskUuid, subtask.uuid, subtask.SlaveNodeUuid, errors.New(string(respBody)))
						mc.AlertSubtaskError(subtask.uuid, subtask.SlaveNodeUuid, err.Error())
						continue
					}

					var status string
					err = json.NewDecoder(resp.Body).Decode(&status)
					if err != nil {
						log.Printf("[SUBTASK_WORKER][TASK | %s][SUBTASK | %s][SLAVE | %s] error: %v\n", subtask.TaskUuid, subtask.uuid, subtask.SlaveNodeUuid, err)
						mc.AlertSubtaskError(subtask.uuid, subtask.SlaveNodeUuid, err.Error())
						continue
					}
					if status == "error" {
						log.Printf("[SUBTASK_WORKER][TASK | %s][SUBTASK | %s][SLAVE | %s] slave have 'error' status\n", subtask.TaskUuid, subtask.uuid, subtask.SlaveNodeUuid)
						mc.AlertSubtaskError(subtask.uuid, subtask.SlaveNodeUuid, fmt.Sprintf("[SUBTASK_WORKER][TASK | %s][SUBTASK | %s][SLAVE | %s] slave have 'error' status", subtask.TaskUuid, subtask.uuid, subtask.SlaveNodeUuid))
					} else if status != "solving" {
						log.Printf("[SUBTASK_WORKER][TASK | %s][SUBTASK | %s][SLAVE | %s] slave dont have 'solving' status\n", subtask.TaskUuid, subtask.uuid, subtask.SlaveNodeUuid)
						mc.AlertSubtaskError(subtask.uuid, subtask.SlaveNodeUuid, fmt.Sprintf("[SUBTASK_WORKER][TASK | %s][SUBTASK | %s][SLAVE | %s] slave dont have 'solving' status", subtask.TaskUuid, subtask.uuid, subtask.SlaveNodeUuid))
					}

				}

			}

		}

	}

}

func (mc *ManagerClient) SetTask(taskCfg TaskConfig) error {
	if v, ok := mc.MasterNodes[taskCfg.masterUUID]; ok {
		v.generatorScript = taskCfg.generatorScript
		v.computeScript = taskCfg.computeScript
		v.taskName = taskCfg.taskName
		v.task = taskCfg.task
		ctx, cancel := context.WithCancel(context.Background())
		v.cancelTask = cancel
		mc.MasterNodes[taskCfg.masterUUID] = v
		mc.taskStatus[taskCfg.masterUUID] = Task{
			MasterUuid: taskCfg.masterUUID,
			Data:       taskCfg.data,
			counter:    0,
		}

		go mc.taskWorker(ctx, taskCfg.masterUUID)
	} else {
		return fmt.Errorf("master node %s not exist", taskCfg.masterUUID)
	}

	return nil
}

func (sd *ManagerClient) RegisterMaster(node model.Node) error {
	sd.MasterNodes[node.Uuid] = &MasterNode{
		Node: node,
	}
	return nil
}

func (sd *ManagerClient) RegisterSlave(node model.Node) error {
	sd.SlaveNodes[node.Uuid] = &SlaveNode{
		Node: node,
		//status: "ok",
	}
	sd.FreeSlaves[node.Uuid] = struct{}{}
	return nil
}

func (sd *ManagerClient) checkMasterHealthWorker() {
	ticker := time.NewTicker(sd.cfg.CheckHealthInterval)
	for {
		select {
		case <-ticker.C:
			var res, ex int
			for uuid, node := range sd.MasterNodes {
				_, err := http.Get(fmt.Sprintf("http://%s%s/health", node.Url, node.PrivatePort))
				if err != nil {
					ex++
					sd.mu.Lock()
					delete(sd.MasterNodes, uuid)
					sd.mu.Unlock()
					log.Println("service disconnected:", node)
				} else {
					res++
				}
			}
			log.Printf("Services online:%d, disconnected:%d", res, ex)

		}
	}
}

func (sd *ManagerClient) checkSlaveHealthWorker() {
	ticker := time.NewTicker(sd.cfg.CheckHealthInterval)
	for {
		select {
		case <-ticker.C:
			var res, ex int
			for uuid, node := range sd.SlaveNodes {
				_, err := http.Get(fmt.Sprintf("http://%s%s/health", node.Url, node.PrivatePort))
				if err != nil {
					ex++
					sd.mu.Lock()
					delete(sd.SlaveNodes, uuid)
					delete(sd.FreeSlaves, uuid)
					delete(sd.WorkSlaves, uuid)
					sd.mu.Unlock()
					log.Println("service disconnected:", node)
				} else {
					res++
				}
			}
			log.Printf("Services online:%d, disconnected:%d", res, ex)

		}
	}
}

func (mc *ManagerClient) CloseTask(uuid string) error {
	if master, ok := mc.MasterNodes[uuid]; ok {
		master.cancelTask()
	} else {
		delete(mc.taskStatus, uuid)
		return fmt.Errorf("master node %s not exist", uuid)
	}
	return nil
}
