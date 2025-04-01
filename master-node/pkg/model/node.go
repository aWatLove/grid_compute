package model

type Node struct {
	UUID        string `json:"UUID"`
	Url         string `json:"Url"`
	PublicPort  string `json:"PublicPort"`
	PrivatePort string `json:"PrivatePort"`
}
