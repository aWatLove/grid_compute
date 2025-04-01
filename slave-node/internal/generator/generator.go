package generator

import (
	"encoding/json"
	"fmt"
	"go.starlark.net/starlark"
	"log"
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

	//globals starlark.StringDict
	taskCh   chan model.ComputeRequest
	resultCh chan model.ComputeRequest
}

func NewGenerator() *Generator {
	return &Generator{}
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

func (g *Generator) taskWorker() {
	for task := range g.taskCh {
		g.status = STATUS_SOLVING
		result, err := g.ComputeTask(task)
		g.SendResult(result, err)
		g.status = STATUS_WAIT_TASK
	}
}

func (g *Generator) SendResult(result interface{}, err error) { //todo отправка manager'у результата

}

func (g *Generator) ComputeTask(task model.ComputeRequest) (interface{}, error) {
	// Конвертируем входные данные в Starlark значение
	data, err := parseInputData(task.Data)
	if err != nil {
		return nil, fmt.Errorf("input data error: %v", err)
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
		return nil, fmt.Errorf("script error: %v", err)
	}

	argsGenerate := starlark.Tuple{
		data,
		starlark.MakeInt(task.Amount),
		starlark.MakeInt(task.Start),
	}

	resultGenerate, err := starlark.Call(threadGenerate, globalsGenerate[task.Generate.FuncName], argsGenerate, nil)
	if err != nil {
		return nil, fmt.Errorf("script error while calling: %v", err)
	}

	builtinsCompute := starlark.StringDict{
		"input_data": resultGenerate,
		"error":      starlark.None,
	}

	threadCompute := &starlark.Thread{
		Name:  "starlark",
		Print: func(_ *starlark.Thread, msg string) { log.Println("[SCRIPT][COMPUTE]", msg) },
	}

	globalsCompute, err := starlark.ExecFile(threadCompute, "compute.star", task.Compute.Script, builtinsCompute)
	if err != nil {
		return nil, fmt.Errorf("script error: %v", err)
	}

	argsCompute := starlark.Tuple{
		data,
		starlark.MakeInt(task.Amount),
		starlark.MakeInt(task.Start),
	}

	result, err := starlark.Call(threadCompute, globalsCompute[task.Compute.FuncName], argsCompute, nil)
	if err != nil {
		return nil, fmt.Errorf("script error while calling: %v", err)
	}

	return convertToGoType(result)
}

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
