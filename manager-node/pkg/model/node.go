package model

type Node struct {
	Uuid        string `json:"UUID"`
	Url         string `json:"Url"`
	PublicPort  string `json:"PublicPort"`
	PrivatePort string `json:"PrivatePort"`
}
