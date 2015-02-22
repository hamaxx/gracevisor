package main

import (
	"io/ioutil"
	"path"
	"time"

	"github.com/hamaxx/gracevisor/deps/yaml.v2"
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

	StopSignal   int           `yaml:"stop_signal"`
	MaxRetries   int           `yaml:"max_retries"`
	StartTimeout time.Duration `yaml:"start_timeout"`
	StopTimeout  time.Duration `yaml:"stop_timeout"`

	InternalHost string `yaml:"internal_host"`
	ExternalHost string `yaml:"external_host"`
	ExternalPort uint32 `yaml:"external_port"`
}

type RpcConfig struct {
	Host string `yaml:"host"`
	Port uint32 `yaml:"port"`
}

type Config struct {
	PortRange *InternalPorts `yaml:"port_range"`
	Apps      []*AppConfig   `yaml:"apps"`
	Rpc       *RpcConfig     `yaml:"rpc"`
}

func ParseConfing(configPath string) (*Config, error) {
	// TODO: validate params, default values, conf.d
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
