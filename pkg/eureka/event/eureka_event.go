package event

import (
	_ "github.com/Shopify/sarama"
)

type Event struct {
	Status  string `json:"status"`
	AppName string `json:"appName"`
	Id      string `json:"id"`
	Version string `json:"version"`
	Uuid    string `json:"uuid"`
	ApiData string `json:"apiData"`
}
