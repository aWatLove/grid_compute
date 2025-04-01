package model

import "encoding/json"

type ComputeRequest struct {
	Data json.RawMessage `json:"data"`
}

type ScriptRequest struct {
	Script      string `json:"function"`
	ComputeFunc string `json:"computeFunc"`
}

type ComputeResponse struct {
	Result interface{} `json:"result"`
	Error  string      `json:"error,omitempty"`
}
