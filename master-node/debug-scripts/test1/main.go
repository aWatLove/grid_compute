package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type ComputeRequest struct {
	Function string      `json:"function"`
	Data     interface{} `json:"data"`
}

func main() {
	// Пример функции и данных
	functionCode := `
def compute(input_data):
    result = input_data * 2 + 5
    return result`
	data := 10

	// Создаем запрос
	req := ComputeRequest{
		Function: functionCode,
		Data:     data,
	}

	// Отправляем на слейв
	resp, err := http.Post(
		"http://localhost:8081/api/v1/compute",
		"application/json",
		bytes.NewBuffer(mustJSON(req)),
	)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// Читаем ответ
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("Response: %s\n", body)
}

func mustJSON(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
