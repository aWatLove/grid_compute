package tasker

import (
	"encoding/json"
	"log"
)

type TaskEngineMock struct {
}

func (t TaskEngineMock) ConfirmSubtaskHandler(message json.RawMessage) {
	log.Println("ConfirmSubtaskHandler:", string(message))
}

func (t TaskEngineMock) DoneTaskHandler() {
	log.Println("DoneTaskHandler")
}

func (t TaskEngineMock) ErrorTaskHandler(err error) {
	log.Println("ErrorTaskHandler:", err)
}
