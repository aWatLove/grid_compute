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
	Amount   uint32          `json:"Amount"`         // кол-во подзадач которое нужно сгенерировать
	Start    uint32          `json:"Start"`          // позиция от которой генерировать данные для просчета
}

type CompleteSubtaskRequest struct {
	SlaveUUID   string          `json:"UUID"`
	SubtaskUUID string          `json:"SubtaskUUID"`
	Status      string          `json:"Status"`
	Data        json.RawMessage `json:"Data"`
}
