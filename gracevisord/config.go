package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/hamaxx/gracevisor/deps/yaml.v2"
)

var configFile = "gracevisor.yaml"

var (
	ErrInvalidPortRange = errors.New("Invalid port range")
)

var (
	defaultPortFrom = 10000
	defaultPortTo   = 11000

	defaultLogFile     = "/var/log/gracevisor/gracevisor.log"
	defaultLogDir      = "/var/log/gracevisor"
	defaultMaxLogSize  = 500
	defaultLogFileMode = os.FileMode(0600)
)

type InternalPortsConfig struct {
	From uint32 `yaml:"from"`
	To   uint32 `yaml:"to"`
}

func (c *InternalPortsConfig) clean(g *Config) error {
	if c.From == 0 && c.To == 0 {
		c.From = uint32(defaultPortFrom)
		c.To = uint32(defaultPortTo)
	}

	if c.From >= c.To {
		return ErrInvalidPortRange
	}

	return nil
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

func (c *AppConfig) clean(g *Config) error {
	// TODO

	if c.StdoutLogFile == "" {
		c.StdoutLogFile = path.Join(g.Logger.ChildLogDir, fmt.Sprintf("app_%s.out", c.Name))
	}
	if c.StderrLogFile == "" {
		c.StderrLogFile = path.Join(g.Logger.ChildLogDir, fmt.Sprintf("app_%s.err", c.Name))
	}

	if err := os.MkdirAll(path.Dir(c.StdoutLogFile), defaultLogFileMode); err != nil {
		return err
	}
	if err := os.MkdirAll(path.Dir(c.StderrLogFile), defaultLogFileMode); err != nil {
		return err
	}

	return nil
}

type RpcConfig struct {
	Host string `yaml:"host"`
	Port uint32 `yaml:"port"`
}

func (c *RpcConfig) clean(g *Config) error {
	// TODO
	return nil
}

type LoggerConfig struct {
	ChildLogDir string `yaml:"child_log_dir"`
	LogFile     string `yaml:"log_file"`
	MaxLogSize  int    `yaml:"max_log_size"`
	MaxLogsKept int    `yaml:"max_logs_kept"`
	MaxLogAge   int    `yaml:"max_log_age"`
}

func (c *LoggerConfig) clean(g *Config) error {
	if c.ChildLogDir == "" {
		c.ChildLogDir = defaultLogDir
	}
	if c.LogFile == "" {
		c.LogFile = defaultLogFile
	}
	if c.MaxLogSize <= 0 {
		c.MaxLogSize = defaultMaxLogSize
	}

	if err := os.MkdirAll(path.Dir(c.LogFile), defaultLogFileMode); err != nil {
		return err
	}

	return nil
}

type Config struct {
	PortRange *InternalPortsConfig `yaml:"port_range"`
	Apps      []*AppConfig         `yaml:"apps"`
	Rpc       *RpcConfig           `yaml:"rpc"`
	Logger    *LoggerConfig        `yaml:"logger"`
}

func (c *Config) clean(g *Config) error {
	if c.PortRange == nil {
		c.PortRange = &InternalPortsConfig{}
	}
	if c.Apps == nil {
		c.Apps = []*AppConfig{}
	}
	if c.Rpc == nil {
		c.Rpc = &RpcConfig{}
	}
	if c.Logger == nil {
		c.Logger = &LoggerConfig{}
	}

	if err := c.PortRange.clean(c); err != nil {
		return err
	}
	if err := c.Rpc.clean(c); err != nil {
		return err
	}
	if err := c.Logger.clean(c); err != nil {
		return err
	}
	for _, app := range c.Apps {
		if err := app.clean(c); err != nil {
			return err
		}
	}
	return nil
}

func ParseConfing(configPath string) (*Config, error) {
	// TODO: validate params, default values, conf.d
	data, err := ioutil.ReadFile(path.Join(configPath, configFile))
	if err != nil {
		return nil, err
	}

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	if err := config.clean(config); err != nil {
		return nil, err
	}

	return config, err
}
