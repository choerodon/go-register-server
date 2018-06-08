package event

import "time"

type jsonTime time.Time

type Event struct {
	Status  string `json:"status"`
	AppName string `json:"appName"`
	Version string `json:"version"`
	InstanceAddress string `json:"instanceAddress"`
	CreateTime      jsonTime  `json:"createTime"`
}
