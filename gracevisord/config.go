package main

import (
	"io/ioutil"
	"path"

	"gopkg.in/yaml.v2"
)

var configFile = "gracevisor.yaml"

type InternalPorts struct {
	From uint32 `yaml:"from:`
	To   uint32 `yaml:"to"`
}

type AppConfig struct {
	Name        string `yaml:"name"`
	Command     string `yaml:"command"`
	Healthcheck string `yaml:"Healthcheck"`
	StopSignal  int    `yaml:"stop_signal"`
	Timeout     int    `yaml:"timeout"`

	InternalHost string `yaml:"internal_host"`
	ExternalHost string `yaml:"external_host"`
	ExternalPort uint32 `yaml:"external_port"`
}

type Config struct {
	Port *InternalPorts `yaml:"port"`
	Apps []*AppConfig   `yaml:"apps"`
}

func ParseConfing(configPath string) (*Config, error) {
	// TODO: validate params, default values
	data, err := ioutil.ReadFile(path.Join(configPath, configFile))
	if err != nil {
		return nil, err
	}

	config := &Config{}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}
	return config, err
}
