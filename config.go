package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type config map[string]interface{}

func (c config) Get(key string) string {
	v, ok := c[key].(string)
	if !ok {
		return ""
	}

	return v
}

func (c config) GetRaw(key string) interface{} {
	v, ok := c[key]
	if !ok {
		return nil
	}

	return v
}

func Config() config {
	f, err := os.Open("config.json")
	if err != nil {
		return config{}
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return config{}
	}

	var c config
	if json.Unmarshal(b, &c) != nil {
		return config{}
	}

	return c
}
