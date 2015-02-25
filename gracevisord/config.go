package main

import (
	"io/ioutil"
	"path"

	"github.com/hamaxx/gracevisor/deps/yaml.v2"
)

var configFile = "gracevisor.yaml"

type InternalPortsConfig struct {
	From uint32 `yaml:"from"`
	To   uint32 `yaml:"to"`
}

type AppConfig struct {
	Name        string `yaml:"name"`
	Command     string `yaml:"command"`
	HealthCheck string `yaml:"healthcheck"`

	StopSignal   string `yaml:"stop_signal"`
	MaxRetries   int    `yaml:"max_retries"`
	StartTimeout int    `yaml:"start_timeout"`
	StopTimeout  int    `yaml:"stop_timeout"`

	InternalHost string `yaml:"internal_host"`
	ExternalHost string `yaml:"external_host"`
	ExternalPort uint32 `yaml:"external_port"`

	StdoutLogFile string `yaml:"stdout_log_file"`
	StderrLogFile string `yaml:"stderr_log_file"`
}

type RpcConfig struct {
	Host string `yaml:"host"`
	Port uint32 `yaml:"port"`
}

type LoggerConfig struct {
	ChildLogDir string `yaml:"child_log_dir"`
	LogFile     string `yaml:"log_file"`
	MaxLogSize  int    `yaml:"max_log_size"`
	MaxLogsKept int    `yaml:"max_logs_kept"`
	MaxLogAge   int    `yaml:"max_log_age"`
}

type Config struct {
	PortRange *InternalPortsConfig `yaml:"port_range"`
	Apps      []*AppConfig         `yaml:"apps"`
	Rpc       *RpcConfig           `yaml:"rpc"`
	Logger    *LoggerConfig        `yaml:"logger"`
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
