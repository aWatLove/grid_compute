package model

import "encoding/json"

type ScriptConfig struct {
	Script   string `json:"Script"`   // код скрипта
	FuncName string `json:"FuncName"` // имя функции которую нужно вызвать
}

type ComputeRequest struct {
	Generate ScriptConfig    `json:"GenerateScript"` // скрипт генерации подзадач из данных
	Compute  ScriptConfig    `json:"ComputeScript"`  // скрипт решения подзадач
	Data     json.RawMessage `json:"Data"`           // данные из которых нужно генерировать
	Amount   int             `json:"Amount"`         // кол-во подзадач которое нужно сгенерировать
	Start    int             `json:"Start"`          // позиция от которой генерировать данные для просчета
}

type ScriptRequest struct {
	Script      string `json:"function"`
	ComputeFunc string `json:"computeFunc"`
}

type ComputeResponse struct {
	Result interface{} `json:"result"`
	Error  string      `json:"error,omitempty"`
}

type Node struct {
	UUID        string `json:"UUID"`
	Url         string `json:"Url"`
	PublicPort  string `json:"PublicPort"`
	PrivatePort string `json:"PrivatePort"`
}
