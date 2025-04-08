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
	MasterUUID      string             `json:"MasterUUID"`
	GeneratorScript model.ScriptConfig `json:"GeneratorScript"`
	ComputeScript   model.ScriptConfig `json:"ComputeScript"`
	Data            json.RawMessage    `json:"Data"`
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
						mc.taskStatus[uuid] = task

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
	mc.mu.Lock()
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
	mc.mu.Unlock()

	task := mc.taskStatus[subtask.TaskUuid]
	mn, ok := mc.MasterNodes[task.MasterUuid]
	if !ok {
		mc.alertTaskError(taskUuid, "master node not found")
	}

	reqBody := model.ComputeRequest{
		UuidSubtask: subtaskUuid,
		Generate:    mn.generatorScript,
		Compute:     mn.computeScript,
		Data:        task.Data,
		Amount:      amount,
		Start:       start,
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

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s%s%s", node.Url, node.PublicPort, "/api/v1/addTask"), bytes.NewReader(data))
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

	reqBody := struct {
		Data json.RawMessage `json:"Data"`
	}{data}

	dataReqBody, err := json.Marshal(reqBody)

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s%s%s", master.Url, master.PublicPort, "/api/v1/subtask/done"), bytes.NewReader(dataReqBody)) //todo checkme чекнуть как порт будет передаваться {"8080"/":8080"}
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

func (mc *ManagerClient) alertTaskError(uuid string, errorStr string) { // отправка уведомления об ошибке мастеру и удаление задачи
	// найти мастера по uuid , отправить ему ошибку по ручке
	_, ok := mc.taskStatus[uuid]
	if !ok {
		log.Printf("[DONE TASK][ERROR] task not found with uuid %s\n", uuid)
	}
	mc.mu.Lock()
	delete(mc.taskStatus, uuid)
	mc.mu.Unlock()

	master, ok := mc.MasterNodes[uuid]
	if !ok {
		log.Printf("[DONE TASK][ERROR] master node not found with uuid %s\n", uuid)
	} else {
		master.cancelTask()
	}

	client := http.Client{}

	data, err := json.Marshal(errorStr)
	if err != nil {
		log.Println(err)
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s%s", master.Url, "/api/v1/task/error"), bytes.NewReader(data))
	if err != nil {
		log.Println(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
	}
	if resp.StatusCode/100 != 2 {
		respBody, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		log.Println(err, string(respBody))
	}

	log.Printf("[TASK ALERT] ALERT!!!! with uuid: %s\n", uuid)

}

func (mc *ManagerClient) doneTask(uuid string) {
	// сделать проверку на то что
	// Проверяем все имеющиейся подзадачи на uuid Главной таски, если они есть, то дожидаемся от них ответа. и только после этого уведомляем мастера о том что задача выполнилась мастера /task/done
	_, ok := mc.taskStatus[uuid]
	if !ok {
		log.Printf("[DONE TASK][ERROR] task not found with uuid %s\n", uuid)
	}
	mc.mu.Lock()
	delete(mc.taskStatus, uuid)
	mc.mu.Unlock()

	master, ok := mc.MasterNodes[uuid]
	if !ok {
		log.Printf("[DONE TASK][ERROR] master node not found with uuid %s\n", uuid)
	}

	master.cancelTask()

	subtasksMap := make(map[string]Subtask)

	for s, subtask := range mc.subtasksStatus {
		if subtask.TaskUuid == uuid {
			subtasksMap[s] = subtask
		}
	}

	for len(subtasksMap) != 0 {
		for s, _ := range subtasksMap {
			if _, ok := mc.subtasksStatus[s]; !ok {
				delete(mc.subtasksStatus, s)
			}
		}
		time.Sleep(5 * time.Second)
	}

	client := http.Client{}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s%s%s", master.Url, master.PublicPort, "/api/v1/task/done"), nil)
	if err != nil {
		log.Println(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
	}
	if resp.StatusCode/100 != 2 {
		respBody, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		log.Println(err, string(respBody))
	}

	log.Printf("[TASK DONE] DONE!!!! with uuid: %s\n", uuid)
}

func (mc *ManagerClient) CompleteSubTask(resp model.CompleteSubtaskRequest) error {
	go func() {
		err := mc.completeSubTask(resp)
		if err != nil {
			log.Println("[COMPLETE_SUBTASK ERROR]:", err)
		}
	}()

	return nil
}

func (mc *ManagerClient) completeSubTask(resp model.CompleteSubtaskRequest) error {
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
	mc.mu.Lock()
	delete(mc.subtasksStatus, resp.SubtaskUUID)
	mc.mu.Unlock()

	mc.mu.Lock()
	task, ok := mc.taskStatus[subtask.TaskUuid]
	mc.mu.Unlock()

	if !ok {
		return errors.New("task not found")
	}

	if resp.Status == "empty" {
		log.Println("TASK is done with uuid: ", subtask.TaskUuid)
		go mc.doneTask(subtask.TaskUuid)
		return nil
	}

	return mc.sendMasterSubTask(resp.Data, task.MasterUuid)
}

// subtaskWorker - воркер чекинга состояния решения подзадач у слейвов
func (mc *ManagerClient) subtaskWorker() { // здесь пингуются сабтаски
	ticker := time.NewTicker(30 * time.Second)

	//client := &http.Client{}

	for {
		select {
		case <-ticker.C:
			mc.mu.Lock()
			if len(mc.subtasksStatus) != 0 {
				mc.mu.Unlock()

				//for _, subtask := range mc.subtasksStatus {
				//
				//	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s%s", subtask.Url, "/api/v1/checkStatus"), nil)
				//	if err != nil {
				//		log.Printf("[SUBTASK_WORKER][TASK | %s][SUBTASK | %s][SLAVE | %s] error: %v\n", subtask.TaskUuid, subtask.uuid, subtask.SlaveNodeUuid, err)
				//		mc.AlertSubtaskError(subtask.uuid, subtask.SlaveNodeUuid, err.Error())
				//		continue
				//	}
				//
				//	resp, err := client.Do(req)
				//	if err != nil {
				//		log.Printf("[SUBTASK_WORKER][TASK | %s][SUBTASK | %s][SLAVE | %s] error: %v\n", subtask.TaskUuid, subtask.uuid, subtask.SlaveNodeUuid, err)
				//		mc.AlertSubtaskError(subtask.uuid, subtask.SlaveNodeUuid, err.Error())
				//		continue
				//	}
				//	if resp.StatusCode/100 != 2 {
				//		respBody, _ := io.ReadAll(resp.Body)
				//		defer resp.Body.Close()
				//		log.Printf("[SUBTASK_WORKER][TASK | %s][SUBTASK | %s][SLAVE | %s] error: %v\n", subtask.TaskUuid, subtask.uuid, subtask.SlaveNodeUuid, errors.New(string(respBody)))
				//		mc.AlertSubtaskError(subtask.uuid, subtask.SlaveNodeUuid, err.Error())
				//		continue
				//	}
				//
				//	respBody, _ := io.ReadAll(resp.Body)
				//	var status = string(respBody)
				//
				//	if err != nil {
				//		log.Printf("[SUBTASK_WORKER][TASK | %s][SUBTASK | %s][SLAVE | %s] error: %v\n", subtask.TaskUuid, subtask.uuid, subtask.SlaveNodeUuid, err)
				//		mc.AlertSubtaskError(subtask.uuid, subtask.SlaveNodeUuid, err.Error())
				//		continue
				//	}
				//	if status == "error" {
				//		log.Printf("[SUBTASK_WORKER][TASK | %s][SUBTASK | %s][SLAVE | %s] slave have 'error' status\n", subtask.TaskUuid, subtask.uuid, subtask.SlaveNodeUuid)
				//		mc.AlertSubtaskError(subtask.uuid, subtask.SlaveNodeUuid, fmt.Sprintf("[SUBTASK_WORKER][TASK | %s][SUBTASK | %s][SLAVE | %s] slave have 'error' status", subtask.TaskUuid, subtask.uuid, subtask.SlaveNodeUuid))
				//	} else if status != "solving" {
				//		log.Printf("[SUBTASK_WORKER][TASK | %s][SUBTASK | %s][SLAVE | %s] slave dont have 'solving' status\n", subtask.TaskUuid, subtask.uuid, subtask.SlaveNodeUuid)
				//		mc.AlertSubtaskError(subtask.uuid, subtask.SlaveNodeUuid, fmt.Sprintf("[SUBTASK_WORKER][TASK | %s][SUBTASK | %s][SLAVE | %s] slave dont have 'solving' status", subtask.TaskUuid, subtask.uuid, subtask.SlaveNodeUuid))
				//	}
				//
				//}

			} else {
				mc.mu.Unlock()
			}

		}

	}

}

func (mc *ManagerClient) SetTask(taskCfg TaskConfig) error {
	if v, ok := mc.MasterNodes[taskCfg.MasterUUID]; ok {
		v.generatorScript = taskCfg.GeneratorScript
		v.computeScript = taskCfg.ComputeScript
		v.taskName = taskCfg.taskName
		v.task = Task{
			MasterUuid: taskCfg.MasterUUID,
			Data:       taskCfg.Data,
			counter:    0,
		}
		ctx, cancel := context.WithCancel(context.Background())
		v.cancelTask = cancel
		mc.MasterNodes[taskCfg.MasterUUID] = v
		mc.taskStatus[taskCfg.MasterUUID] = Task{
			MasterUuid: taskCfg.MasterUUID,
			Data:       taskCfg.Data,
			counter:    0,
		}

		go mc.taskWorker(ctx, taskCfg.MasterUUID)
	} else {
		return fmt.Errorf("master node %s not exist", taskCfg.MasterUUID)
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
			log.Printf("[MASTERS] Services online:%d, disconnected:%d", res, ex)

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
			log.Printf("[SLAVES] Services online:%d, disconnected:%d", res, ex)

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
