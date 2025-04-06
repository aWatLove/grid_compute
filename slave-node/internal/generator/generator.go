package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go.starlark.net/starlark"
	"io/ioutil"
	"log"
	"net/http"
	"slave-node/internal/config"
	"slave-node/pkg/model"
)

//type ScriptConfig struct {
//	Script   string `json:"Script"`
//	FuncName string `json:"FuncName"`
//}

var statusStr = []string{
	"waiting task",
	"solving",
	"error",
}

const (
	STATUS_WAIT_TASK uint8 = iota
	STATUS_SOLVING
	STATUS_ERROR
)

type Generator struct {
	status uint8
	cfg    *config.Config

	taskCh chan model.ComputeRequest
}

func NewGenerator(cfg *config.Config) *Generator {
	return &Generator{
		status: 0,
		cfg:    cfg,
		taskCh: make(chan model.ComputeRequest),
	}
}

func (g *Generator) AddTask(task model.ComputeRequest) error {
	if g.status != STATUS_WAIT_TASK {
		return fmt.Errorf("NODE has status: %s", statusStr[g.status])
	}

	g.taskCh <- task
	return nil
}

func (g *Generator) CheckStatus() string {
	return statusStr[g.status]
}

// Обновленный обработчик задач
func (g *Generator) taskWorker() {
	for task := range g.taskCh {
		g.status = STATUS_SOLVING
		data, status, err := g.ComputeTask(task)
		if err != nil {
			status = "error"
			err = g.SendAlert(ErrorSubtaskReq{
				SlaveUUID:   g.cfg.UUID,
				SubtaskUUID: task.UuidSubtask,
				Error:       err.Error(),
			})
			g.status = STATUS_ERROR

		} else {
			err = g.SendResult(task, data, status)
			if err != nil {
				g.status = STATUS_ERROR
			} else {
				g.status = STATUS_WAIT_TASK
			}

		}

	}
}

// Обновленный метод SendResult
func (g *Generator) SendResult(task model.ComputeRequest, data interface{}, status string) error {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal data: %v", err)
		return fmt.Errorf("Failed to marshal data: %v", err)
	}

	result := CompleteSubtaskRequest{
		SlaveUUID:   g.cfg.UUID,       // Предполагается наличие поля
		SubtaskUUID: task.UuidSubtask, // Из исходной задачи
		Status:      status,
		Data:        json.RawMessage(dataBytes),
	}

	client := http.Client{}

	dataRes, err := json.Marshal(result)
	if err != nil {
		log.Printf("Failed to marshal data: %v", err)
		return fmt.Errorf("Failed to marshal data: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s%s", g.cfg.ManagerURL, "/api/v1/subtask/complete"), bytes.NewReader(dataRes))
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return fmt.Errorf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to send request: %v", err)
		return fmt.Errorf("Failed to send request: %v", err)
	}

	if resp.StatusCode/100 != 2 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Failed to read response body: %v", err)
			return fmt.Errorf("Failed to read response body: %v", err)
		}
		defer resp.Body.Close()
		log.Printf("Response body: %s", string(body))
		return fmt.Errorf("Failed to send request: %s", string(body))
	}

	return nil
}

type ErrorSubtaskReq struct {
	SlaveUUID   string `json:"SlaveUUID"`
	SubtaskUUID string `json:"SubtaskUUID"`
	Error       string `json:"Error"`
}

// SendAlert
func (g *Generator) SendAlert(reqBody ErrorSubtaskReq) error {

	client := http.Client{}

	dataRes, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("Failed to marshal data: %v", err)
		return fmt.Errorf("Failed to marshal data: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s%s", g.cfg.ManagerURL, "/api/v1/subtask/complete"), bytes.NewReader(dataRes))
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return fmt.Errorf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to send request: %v", err)
		return fmt.Errorf("Failed to send request: %v", err)
	}

	if resp.StatusCode/100 != 2 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Failed to read response body: %v", err)
			return fmt.Errorf("Failed to read response body: %v", err)
		}
		defer resp.Body.Close()
		log.Printf("Response body: %s", string(body))
		return fmt.Errorf("Failed to send request: %s", string(body))
	}

	return nil
}

type CompleteSubtaskRequest struct {
	SlaveUUID   string          `json:"UUID"`
	SubtaskUUID string          `json:"SubtaskUUID"`
	Status      string          `json:"Status"`
	Data        json.RawMessage `json:"Data"`
}

//func (g *Generator) ComputeTask(task model.ComputeRequest) (interface{}, error) {
//	// Конвертируем входные данные в Starlark значение
//	data, err := parseInputData(task.Data)
//	if err != nil {
//		return nil, fmt.Errorf("input data error: %v", err)
//	}
//
//	// Создаем окружение с входными данными
//	builtinsGenerate := starlark.StringDict{
//		"input_data": data,
//		"amount":     starlark.MakeInt(task.Amount),
//		"start":      starlark.MakeInt(task.Start),
//		"error":      starlark.None,
//	}
//
//	// Выполняем скрипт
//	threadGenerate := &starlark.Thread{
//		Name:  "starlark",
//		Print: func(_ *starlark.Thread, msg string) { log.Println("[SCRIPT][GENERATE]", msg) },
//	}
//
//	globalsGenerate, err := starlark.ExecFile(threadGenerate, "compute.star", task.Generate.Script, builtinsGenerate)
//	if err != nil {
//		return nil, fmt.Errorf("script error: %v", err)
//	}
//
//	argsGenerate := starlark.Tuple{
//		data,
//		starlark.MakeInt(task.Amount),
//		starlark.MakeInt(task.Start),
//	}
//
//	resultGenerate, err := starlark.Call(threadGenerate, globalsGenerate[task.Generate.FuncName], argsGenerate, nil)
//	if err != nil {
//		return nil, fmt.Errorf("script error while calling: %v", err)
//	}
//
//	builtinsCompute := starlark.StringDict{
//		"input_data": resultGenerate,
//		"error":      starlark.None,
//	}
//
//	threadCompute := &starlark.Thread{
//		Name:  "starlark",
//		Print: func(_ *starlark.Thread, msg string) { log.Println("[SCRIPT][COMPUTE]", msg) },
//	}
//
//	globalsCompute, err := starlark.ExecFile(threadCompute, "compute.star", task.Compute.Script, builtinsCompute)
//	if err != nil {
//		return nil, fmt.Errorf("script error: %v", err)
//	}
//
//	argsCompute := starlark.Tuple{
//		data,
//		starlark.MakeInt(task.Amount),
//		starlark.MakeInt(task.Start),
//	}
//
//	result, err := starlark.Call(threadCompute, globalsCompute[task.Compute.FuncName], argsCompute, nil)
//	if err != nil {
//		return nil, fmt.Errorf("script error while calling: %v", err)
//	}
//
//	return convertToGoType(result)
//}

// ComputeTask возвращает данные, статус и ошибку
func (g *Generator) ComputeTask(task model.ComputeRequest) (interface{}, string, error) {
	// ... [парсинг входных данных] ...
	// Конвертируем входные данные в Starlark значение
	data, err := parseInputData(task.Data)
	if err != nil {
		return nil, "error", fmt.Errorf("input data error: %v", err)
	}

	// Создаем окружение с входными данными
	builtinsGenerate := starlark.StringDict{
		"input_data": data,
		"amount":     starlark.MakeInt(task.Amount),
		"start":      starlark.MakeInt(task.Start),
		"error":      starlark.None,
	}

	// Выполняем скрипт
	threadGenerate := &starlark.Thread{
		Name:  "starlark",
		Print: func(_ *starlark.Thread, msg string) { log.Println("[SCRIPT][GENERATE]", msg) },
	}

	globalsGenerate, err := starlark.ExecFile(threadGenerate, "compute.star", task.Generate.Script, builtinsGenerate)
	if err != nil {
		return nil, "error", fmt.Errorf("script error: %v", err)
	}

	argsGenerate := starlark.Tuple{
		data,
		starlark.MakeInt(task.Amount),
		starlark.MakeInt(task.Start),
	}

	// Выполнение скрипта Generate
	resultGenerate, err := starlark.Call(threadGenerate, globalsGenerate[task.Generate.FuncName], argsGenerate, nil)
	if err != nil {
		return nil, "error", fmt.Errorf("generate script error: %v", err)
	}

	// Извлекаем статус и данные из Generate
	dataGenerate, statusGenerate, err := extractStatusAndData(resultGenerate)
	if err != nil {
		return nil, "error", fmt.Errorf("generate result parsing error: %v", err)
	}

	switch statusGenerate {
	case "error":
		return nil, "error", nil
	case "empty":
		goData, err := convertToGoType(dataGenerate)
		if err != nil {
			return nil, "error", err
		}
		return goData, "empty", nil
	case "ok":
		// Продолжаем выполнение
	default:
		return nil, "error", fmt.Errorf("unknown generate status: %s", statusGenerate)
	}

	builtinsCompute := starlark.StringDict{
		"input_data": dataGenerate,
		"error":      starlark.None,
	}

	threadCompute := &starlark.Thread{
		Name:  "starlark",
		Print: func(_ *starlark.Thread, msg string) { log.Println("[SCRIPT][COMPUTE]", msg) },
	}

	globalsCompute, err := starlark.ExecFile(threadCompute, "compute.star", task.Compute.Script, builtinsCompute)
	if err != nil {
		return nil, "error", fmt.Errorf("script error: %v", err)
	}

	argsCompute := starlark.Tuple{
		dataGenerate,
	}

	// Выполнение скрипта Compute
	resultCompute, err := starlark.Call(threadCompute, globalsCompute[task.Compute.FuncName], argsCompute, nil)
	if err != nil {
		return nil, "error", fmt.Errorf("compute script error: %v", err)
	}

	// Извлекаем статус и данные из Compute
	dataCompute, statusCompute, err := extractStatusAndData(resultCompute)
	if err != nil {
		return nil, "error", fmt.Errorf("compute result parsing error: %v", err)
	}

	// Конвертируем данные в Go-тип
	goData, err := convertToGoType(dataCompute)
	if err != nil {
		return nil, "error", err
	}

	return goData, statusCompute, nil
}

//func (g *Generator) ComputeTask(task model.ComputeRequest) (interface{}, string, error) {
//	// Конвертируем входные данные
//	data, err := parseInputData(task.Data)
//	if err != nil {
//		return nil, "error", fmt.Errorf("input data error: %v", err)
//	}
//
//	// Выполняем Generate скрипт
//	resultGenerate, statusGenerate, err := g.executeGenerateScript(task, data)
//	if statusGenerate != "ok" || err != nil {
//		return nil, statusGenerate, err
//	}
//
//	// Подготавливаем входные данные для Compute
//	computeInput := starlark.NewDict(2)
//	computeInput.SetKey(starlark.String("matrix"), data)
//	computeInput.SetKey(starlark.String("routes"), resultGenerate)
//
//	// Выполняем Compute скрипт
//	resultCompute, statusCompute, err := g.executeComputeScript(task, computeInput)
//	if statusCompute != "ok" || err != nil {
//		return nil, statusCompute, err
//	}
//
//	// Конвертируем финальный результат
//	goData, err := convertToGoType(resultCompute)
//	if err != nil {
//		return nil, "error", fmt.Errorf("result conversion error: %v", err)
//	}
//
//	return goData, "ok", nil
//}
//
//func (g *Generator) executeGenerateScript(task model.ComputeRequest, data starlark.Value) (starlark.Value, string, error) {
//	builtins := starlark.StringDict{
//		"input_data": data,
//		"amount":     starlark.MakeInt(task.Amount),
//		"start":      starlark.MakeInt(task.Start),
//		"error":      starlark.None,
//	}
//
//	thread := &starlark.Thread{Name: "generate"}
//	globals, err := starlark.ExecFile(thread, "generate.star", task.Generate.Script, builtins)
//	if err != nil {
//		return nil, "error", err
//	}
//
//	args := starlark.Tuple{data, starlark.MakeInt(task.Amount), starlark.MakeInt(task.Start)}
//	result, err := starlark.Call(thread, globals[task.Generate.FuncName], args, nil)
//	if err != nil {
//		return nil, "error", err
//	}
//
//	return extractStatusAndData(result)
//}
//
//func (g *Generator) executeComputeScript(task model.ComputeRequest, input *starlark.Dict) (starlark.Value, string, error) {
//	builtins := starlark.StringDict{
//		"input_data": input,
//		"error":      starlark.None,
//	}
//
//	thread := &starlark.Thread{Name: "compute"}
//	globals, err := starlark.ExecFile(thread, "compute.star", task.Compute.Script, builtins)
//	if err != nil {
//		return nil, "error", err
//	}
//
//	result, err := starlark.Call(thread, globals[task.Compute.FuncName], starlark.Tuple{input}, nil)
//	if err != nil {
//		return nil, "error", err
//	}
//
//	return extractStatusAndData(result)
//}

func parseInputData(data json.RawMessage) (starlark.Value, error) {
	var rawData interface{}
	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, err
	}
	return toStarlarkValue(rawData)
}

func toStarlarkValue(v interface{}) (starlark.Value, error) {
	switch v := v.(type) {
	case nil:
		return starlark.None, nil
	case bool:
		return starlark.Bool(v), nil
	case int:
		return starlark.MakeInt(v), nil
	case float64:
		return starlark.Float(v), nil
	case string:
		return starlark.String(v), nil
	case []interface{}:
		elems := make([]starlark.Value, len(v))
		for i, elem := range v {
			val, err := toStarlarkValue(elem)
			if err != nil {
				return nil, err
			}
			elems[i] = val
		}
		return starlark.NewList(elems), nil
	case map[string]interface{}:
		dict := starlark.NewDict(len(v))
		for key, val := range v {
			starlarkVal, err := toStarlarkValue(val)
			if err != nil {
				return nil, err
			}
			if err := dict.SetKey(starlark.String(key), starlarkVal); err != nil {
				return nil, err
			}
		}
		return dict, nil
	default:
		return nil, fmt.Errorf("unsupported type: %T", v)
	}
}

func convertToGoType(v starlark.Value) (interface{}, error) {
	switch v := v.(type) {
	case starlark.NoneType:
		return nil, nil
	case starlark.Bool:
		return bool(v), nil
	case starlark.Int:
		if i, ok := v.Int64(); ok {
			return i, nil
		}
		return nil, fmt.Errorf("integer out of range")
	case starlark.Float:
		return float64(v), nil
	case starlark.String:
		return string(v), nil
	case *starlark.List:
		var result []interface{}
		iter := v.Iterate()
		defer iter.Done()
		var elem starlark.Value
		for iter.Next(&elem) {
			goVal, err := convertToGoType(elem)
			if err != nil {
				return nil, err
			}
			result = append(result, goVal)
		}
		return result, nil
	case *starlark.Dict:
		result := make(map[string]interface{})
		for _, key := range v.Keys() {
			keyStr, ok := key.(starlark.String)
			if !ok {
				return nil, fmt.Errorf("non-string key in dict")
			}
			val, _, _ := v.Get(key)
			goVal, err := convertToGoType(val)
			if err != nil {
				return nil, err
			}
			result[string(keyStr)] = goVal
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported Starlark type: %T", v)
	}
}

// extractStatusAndData извлекает статус и данные из результата Starlark
func extractStatusAndData(result starlark.Value) (data starlark.Value, status string, err error) {
	switch res := result.(type) {
	case *starlark.Dict:
		// Извлекаем статус из словаря
		statusVal, found, err := res.Get(starlark.String("status"))
		if err != nil || !found {
			return nil, "", fmt.Errorf("status not found in result dict")
		}
		statusStr, ok := statusVal.(starlark.String)
		if !ok {
			return nil, "", fmt.Errorf("status is not a string")
		}
		status = string(statusStr)

		// Извлекаем данные из словаря
		dataVal, found, err := res.Get(starlark.String("data"))
		if err != nil || !found {
			return nil, "", fmt.Errorf("data not found in result dict")
		}
		data = dataVal
		return data, status, nil

	case starlark.Tuple:
		if res.Len() != 2 {
			return nil, "", fmt.Errorf("expected tuple of length 2")
		}
		// Статус - первый элемент, данные - второй
		statusVal, ok := res.Index(0).(starlark.String)
		if !ok {
			return nil, "", fmt.Errorf("status in tuple is not a string")
		}
		status = string(statusVal)
		data = res.Index(1)
		return data, status, nil

	default:
		// Если результат не содержит статуса, считаем его "ok"
		return result, "ok", nil
	}
}
