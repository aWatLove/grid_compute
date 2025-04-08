package model

import "encoding/json"

type ComputeRequest struct {
	Data json.RawMessage `json:"data"`
}

type ScriptConfig struct {
	Script   string `json:"Script"`   // код скрипта
	FuncName string `json:"FuncName"` // имя функции которую нужно вызвать
}

type ComputeResponse struct {
	Result interface{} `json:"result"`
	Error  string      `json:"error,omitempty"`
}

type TaskConfig struct {
	MasterUUID      string          `json:"MasterUUID"`
	GeneratorScript ScriptConfig    `json:"GeneratorScript"`
	ComputeScript   ScriptConfig    `json:"ComputeScript"`
	Data            json.RawMessage `json:"Data"`
}
