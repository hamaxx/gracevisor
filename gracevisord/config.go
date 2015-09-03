package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"

	"github.com/hamaxx/gracevisor/deps/yaml.v2"
)

var (
	ErrInvalidPortRange  = errors.New("Invalid port range")
	ErrNameRequired      = errors.New("Name must be specified for app")
	ErrCommandRequired   = errors.New("Command must be specified for app")
	ErrPortBadgeRequired = errors.New("App must have {port} in command or environment")
	ErrInvalidStopSignal = errors.New("Invalid stop signal")
	ErrInvalidUserId     = errors.New("invalid user id format")
)

const (
	configFile = "gracevisor.yaml"

	defaultPortFrom = uint16(10000)
	defaultPortTo   = uint16(11000)

	defaultHost         = "localhost"
	defaultRpcPort      = uint16(9001)
	defaultExternalPort = uint16(8080)

	defaultStopSignal = "TERM"
	defaultMaxRetries = 5

	defaultLogFileName = "gracevisor.log"
	defaultLogDir      = "/var/log/gracevisor"
	defaultMaxLogSize  = 500
	defaultLogFileMode = os.FileMode(0600)
	defaultLogDirMode  = os.FileMode(0744)
)

type UserConfig struct {
	UserName string `yaml:"username"`
	// GroupName string `yaml:"groupname"` TODO when os package will support group lookup

	Uid uint32
}

func (c *UserConfig) clean(g *Config) error {
	if c.UserName == "" {
		return nil
	}

	user, err := user.Lookup(c.UserName)
	if err != nil {
		return err
	}

	uid, err := strconv.ParseUint(user.Uid, 10, 32)
	if err != nil {
		return ErrInvalidUserId
	}

	c.Uid = uint32(uid)

	return nil
}

type InternalPortsConfig struct {
	From uint16 `yaml:"from"`
	To   uint16 `yaml:"to"`
}

func (c *InternalPortsConfig) clean(g *Config) error {
	if c.From == 0 && c.To == 0 {
		c.From = defaultPortFrom
		c.To = defaultPortTo
	}

	if c.From >= c.To {
		return ErrInvalidPortRange
	}

	return nil
}

type AppConfig struct {
	Name        string   `yaml:"name"`
	Command     string   `yaml:"command"`
	Environment []string `yaml:"environment"`
	Directory   string   `yaml:"directory"`
	HealthCheck string   `yaml:"healthcheck"`

	StopSignal     os.Signal
	StopSignalName string `yaml:"stop_signal"`
	MaxRetries     int    `yaml:"max_retries"`
	StartTimeout   int    `yaml:"start_timeout"`
	StopTimeout    int    `yaml:"stop_timeout"`

	InternalHost string `yaml:"internal_host"`
	ExternalHost string `yaml:"external_host"`
	ExternalPort uint16 `yaml:"external_port"`

	Logger *LoggerConfig `yaml:"logger"`
	User   *UserConfig   `yaml:"user"`
}

func (c *AppConfig) clean(g *Config) error {
	if c.Name == "" {
		return ErrNameRequired
	}
	if c.Command == "" {
		return ErrCommandRequired
	}

	if !c.hasPortBadge() {
		return ErrPortBadgeRequired
	}

	if c.StopSignalName == "" {
		c.StopSignalName = defaultStopSignal
	}
	signal, ok := Signals[c.StopSignalName]
	if !ok {
		return ErrInvalidStopSignal
	}
	c.StopSignal = signal

	if c.MaxRetries == 0 {
		c.MaxRetries = defaultMaxRetries
	}

	if c.InternalHost == "" {
		c.InternalHost = defaultHost
	}
	if c.ExternalHost == "" {
		c.ExternalHost = defaultHost
	}

	if c.ExternalPort == 0 {
		c.ExternalPort = defaultExternalPort
	}

	if c.Logger == nil {
		c.Logger = &LoggerConfig{
			LogDir:      g.Logger.LogDir,
			MaxLogSize:  g.Logger.MaxLogSize,
			MaxLogsKept: g.Logger.MaxLogsKept,
			MaxLogAge:   g.Logger.MaxLogAge,
		}
	}
	if err := c.Logger.appClean(g, c); err != nil {
		return err
	}

	if c.User == nil {
		c.User = &UserConfig{}
		if g.User != nil {
			c.User.UserName = g.User.UserName
		}
	}
	if err := c.User.clean(g); err != nil {
		return err
	}

	return nil
}

func (c *AppConfig) hasPortBadge() bool {
	if strings.Contains(c.Command, PortBadge) {
		return true
	}

	for _, env := range c.Environment {
		if strings.Contains(env, PortBadge) {
			return true
		}
	}

	return false
}

type RpcConfig struct {
	Host string `yaml:"host"`
	Port uint16 `yaml:"port"`
}

func (c *RpcConfig) clean(g *Config) error {
	if c.Host == "" {
		c.Host = defaultHost
	}

	if c.Port == 0 {
		c.Port = defaultRpcPort
	}

	return nil
}

type LoggerConfig struct {
	LogDir        string `yaml:"log_dir"`
	LogFile       string `yaml:"log_file"`
	StdoutLogFile string `yaml:"stdout_log_file"`
	StderrLogFile string `yaml:"stderr_log_file"`

	MaxLogSize  int `yaml:"max_log_size"`
	MaxLogsKept int `yaml:"max_logs_kept"`
	MaxLogAge   int `yaml:"max_log_age"`
}

func (c *LoggerConfig) globalClean(g *Config) error {
	if c.LogDir == "" {
		c.LogDir = defaultLogDir
	}
	if c.LogFile == "" {
		c.LogFile = path.Join(c.LogDir, defaultLogFileName)
	}

	if c.MaxLogSize <= 0 {
		c.MaxLogSize = defaultMaxLogSize
	}

	if err := os.MkdirAll(path.Dir(c.LogFile), defaultLogDirMode); err != nil {
		return err
	}

	return nil
}

func (c *LoggerConfig) appClean(g *Config, a *AppConfig) error {
	if c.LogDir == "" {
		c.LogDir = g.Logger.LogDir
	}

	if c.StdoutLogFile == "" {
		if c.LogFile != "" {
			c.StdoutLogFile = c.LogFile
		} else {
			c.StdoutLogFile = path.Join(c.LogDir, fmt.Sprintf("app_%s.out", a.Name))
		}
	}
	if c.StderrLogFile == "" {
		if c.LogFile != "" {
			c.StderrLogFile = c.LogFile
		} else {
			c.StderrLogFile = path.Join(c.LogDir, fmt.Sprintf("app_%s.err", a.Name))
		}
	}

	if c.MaxLogSize <= 0 {
		c.MaxLogSize = g.Logger.MaxLogSize
	}
	if c.MaxLogsKept == 0 {
		c.MaxLogsKept = g.Logger.MaxLogsKept
	}
	if c.MaxLogAge == 0 {
		c.MaxLogAge = g.Logger.MaxLogAge
	}

	if err := os.MkdirAll(path.Dir(c.StdoutLogFile), defaultLogDirMode); err != nil {
		return err
	}
	if err := os.MkdirAll(path.Dir(c.StderrLogFile), defaultLogDirMode); err != nil {
		return err
	}

	return nil
}

type Config struct {
	PortRange *InternalPortsConfig `yaml:"port_range"`
	Apps      []*AppConfig         `yaml:"apps"`
	Rpc       *RpcConfig           `yaml:"rpc"`
	Logger    *LoggerConfig        `yaml:"logger"`
	User      *UserConfig          `yaml:"user"`
	Include   []string             `yaml:"apps_include"`
}

func (c *Config) clean(g *Config) error {
	if c.PortRange == nil {
		c.PortRange = &InternalPortsConfig{}
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
	if err := c.Logger.globalClean(c); err != nil {
		return err
	}
	if c.User != nil {
		if err := c.User.clean(c); err != nil {
			return err
		}
	}

	usedPorts := make(map[uint16]bool)
	usedNames := make(map[string]bool)
	for _, app := range c.Apps {
		if err := app.clean(c); err != nil {
			return fmt.Errorf("%s: %s", app.Name, err)
		}

		_, used := usedPorts[app.ExternalPort]
		if used {
			return fmt.Errorf("%s: Cannot use duplicate external port %d", app.Name, app.ExternalPort)
		}
		usedPorts[app.ExternalPort] = true

		_, used = usedNames[app.Name]
		if used {
			return fmt.Errorf("%s: Cannot use duplicate app name %s", app.Name, app.Name)
		}
		usedNames[app.Name] = true
	}
	return nil
}

func (c *Config) include(inc string) error {
	fi, err := os.Stat(inc)
	if err != nil {
		return err
	}

	if fi.IsDir() {
		files, err := ioutil.ReadDir(inc)
		if err != nil {
			return fmt.Errorf("apps_include: %s", err)
		}

		for _, file := range files {
			if !file.IsDir() {
				if err := c.includeFile(path.Join(inc, file.Name())); err != nil {
					return err
				}
			}
		}
	} else {
		if err := c.includeFile(inc); err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) includeFile(fn string) error {
	if path.Base(fn) == configFile {
		return nil
	}

	if !strings.HasSuffix(fn, ".yaml") {
		return nil
	}

	data, err := ioutil.ReadFile(fn)
	if err != nil {
		return fmt.Errorf("%s: %s", fn, err)
	}

	app := &AppConfig{}
	if err := yaml.Unmarshal(data, app); err != nil {
		return fmt.Errorf("%s: %s", fn, err)
	}

	c.Apps = append(c.Apps, app)

	return nil
}

func ParseConfing(configPath string) (*Config, error) {
	fn := path.Join(configPath, configFile)
	data, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, err
	}

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("%s: %s", fn, err)
	}

	for _, inc := range config.Include {
		if err := config.include(inc); err != nil {
			return nil, err
		}
	}

	if err := config.clean(config); err != nil {
		return nil, err
	}

	return config, err
}
