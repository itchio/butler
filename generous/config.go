package main

type Config struct {
	Input InputConfig
}

type InputConfig struct {
	Package      string         `json:"package"`
	MessagesName string         `json:"messagesName"`
	Targets      []TargetConfig `json:"targets"`
}

type TargetConfig struct {
	Package string   `json:"package"`
	Files   []string ` json:"files"`
}
