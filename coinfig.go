package main

import (
	"encoding/json"
	"os"
)

var config struct {
	Token string `json:"token"`
	AppID string `json:"app_id"`
	Proxy string `json:"proxy"`
}

func loadConfig(filename string) error {
	if file, err := os.Open(filename); err == nil {
		defer file.Close()
		return json.NewDecoder(file).Decode(&config)
	} else {
		return err
	}
}
