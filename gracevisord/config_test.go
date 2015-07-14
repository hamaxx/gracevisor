package main

import (
	"os"
	"os/user"
	"path"
	"strconv"
	"syscall"
	"testing"
)

func TestUserClean(t *testing.T) {
	currentUser, _ := user.Current()

	userConfig := &UserConfig{
		UserName: currentUser.Username,
	}
	if err := userConfig.clean(nil); err != nil {
		t.Error("User config clean fails for valid user:", err)
	}

	currentUserId, _ := strconv.Atoi(currentUser.Uid)
	if userConfig.Uid != currentUserId {
		t.Error("Incorrect user id")
	}

	userConfig.UserName = ""
	if err := userConfig.clean(nil); err != nil {
		t.Error("UserConfig.clean fails for empty username:", err)
	}

	userConfig.UserName = "ThisShouldNotBeAValidUser"
	if err := userConfig.clean(nil); err == nil {
		t.Error("UserConfig.clean does not fail for invalid user.")
	}
}

func TestInternalPortsClean(t *testing.T) {
	internalPortsConfig := &InternalPortsConfig{}

	if err := internalPortsConfig.clean(nil); err != nil {
		t.Error("Internal ports config fails for empty setting:", err)
	}
	if internalPortsConfig.To != defaultPortTo || internalPortsConfig.From != defaultPortFrom {
		t.Error("Incorrect default values set")
	}

	internalPortsConfig.From = 6000
	internalPortsConfig.To = 5000
	if internalPortsConfig.clean(nil) != ErrInvalidPortRange {
		t.Error("Invalid port range does not fail clean")
	}

	internalPortsConfig.From = 5000
	internalPortsConfig.To = 5000
	if internalPortsConfig.clean(nil) != ErrInvalidPortRange {
		t.Error("Zero sized port range does not fail clean")
	}

	internalPortsConfig.From = 5000
	internalPortsConfig.To = 6000
	if err := internalPortsConfig.clean(nil); err != nil {
		t.Error("Valid port range fails clean:", err)
	}
}

func loggersEqual(l1 *LoggerConfig, l2 *LoggerConfig) bool {
	return l1.LogDir == l2.LogDir &&
		l1.MaxLogSize == l2.MaxLogSize &&
		l1.MaxLogsKept == l2.MaxLogsKept &&
		l1.MaxLogAge == l2.MaxLogAge
}

func TestAppClean(t *testing.T) {
	config := &Config{
		Logger: &LoggerConfig{
			LogDir:      "/tmp/log-test/",
			MaxLogSize:  100,
			MaxLogsKept: -1,
			MaxLogAge:   -1,
		},
		User: &UserConfig{
			UserName: "root",
		},
	}
	appConfig := &AppConfig{
		Name:    "demo",
		Command: "../demoapp/demoapp --port={port}",
	}

	if err := appConfig.clean(config); err != nil {
		t.Error("Minimal app config clean fails:", err)
	}

	if appConfig.StopSignal != syscall.SIGTERM {
		t.Error("Incorrect default stop signal set")
	}
	if appConfig.MaxRetries != defaultMaxRetries {
		t.Error("Incorrect default max retries set:", appConfig.MaxRetries)
	}
	if appConfig.InternalHost != defaultHost {
		t.Error("Incorrect default internal host set:", appConfig.InternalHost)
	}
	if appConfig.ExternalHost != defaultHost {
		t.Error("Incorrect default external host set:", appConfig.ExternalHost)
	}
	if appConfig.ExternalPort != defaultExternalPort {
		t.Error("Incorrect default external port set:", appConfig.ExternalPort)
	}

	if appConfig.Logger == config.Logger {
		t.Error("Logger should be new instance not pointer copy")
	}
	if !loggersEqual(appConfig.Logger, config.Logger) {
		t.Error("Logger values not copied correctly")
	}

	if appConfig.User == config.User {
		t.Error("User should be new instance not pointer copy")
	}
	if appConfig.User.UserName != config.User.UserName {
		t.Error("User should have username copied from global config")
	}

	appConfig.Name = ""
	if appConfig.clean(config) != ErrNameRequired {
		t.Error("AppConfig.clean should fail when no name")
	}
	appConfig.Name = "demo"

	appConfig.Command = ""
	if appConfig.clean(config) != ErrCommandRequired {
		t.Error("AppConfig.clean should fail when no command")
	}

	appConfig.Command = "../demoapp/demoapp"
	if appConfig.clean(config) != ErrPortBadgeRequired {
		t.Error("AppConfig.clean should fail when no {port} badge")
	}
	appConfig.Command = "../demoapp/demoapp --port={port}"

	appConfig.StopSignalName = "INT"
	if err := appConfig.clean(config); err != nil {
		t.Error("AppConfig.clean fails with custom signal name:", err)
	}
	if appConfig.StopSignal != syscall.SIGINT {
		t.Error("Incorrect stop signal name to signal conversion")
	}

	appConfig.StopSignalName = "ThisIsNotASignalName"
	if appConfig.clean(config) != ErrInvalidStopSignal {
		t.Error("AppConfig.clean should fail with invalid signal name")
	}
}

func TestAppHasPortBadge(t *testing.T) {
	appConfig := &AppConfig{
		Command: "../demoapp/demoapp --port={port}",
	}
	if !appConfig.hasPortBadge() {
		t.Error("AppConfig.hasPortBadge fails with port badge in command")
	}

	appConfig.Command = "../demoapp/demoapp"
	appConfig.Environment = []string{"PORT={port}"}
	if !appConfig.hasPortBadge() {
		t.Error("AppConfig.hasPortBadge fails with port badge in environment")
	}

	appConfig.Environment = []string{"PORT=80"}
	if appConfig.hasPortBadge() {
		t.Error("AppConfig.hasPortBadge should fail with no port badge")
	}
}

func TestRpcClean(t *testing.T) {
	rpcConfig := &RpcConfig{}
	if err := rpcConfig.clean(nil); err != nil {
		t.Error("RpcConfig.clean fails with for empty setting:", err)
	}

	if rpcConfig.Host != defaultHost {
		t.Error("Incorrect default rpc host set:", rpcConfig.Host)
	}
	if rpcConfig.Port != defaultRpcPort {
		t.Error("Incorrect default rpc port set:", rpcConfig.Port)
	}

	rpcConfig.Host = "test.com"
	rpcConfig.Port = 123
	if err := rpcConfig.clean(nil); err != nil {
		t.Error("RpcConfig.clean fails with valid settings:", err)
	}
}

func TestLoggerGlobalClean(t *testing.T) {
	loggerConfig := &LoggerConfig{}
	loggerConfig.globalClean(nil)

	if loggerConfig.LogDir != defaultLogDir {
		t.Error("Incorrect default log dir set:", loggerConfig.LogDir)
	}
	if loggerConfig.LogFile != path.Join(defaultLogDir, defaultLogFileName) {
		t.Error("Incorrect default log file set:", loggerConfig.LogFile)
	}
	if loggerConfig.MaxLogSize != defaultMaxLogSize {
		t.Error("Incorrect default max log size set:", loggerConfig.MaxLogSize)
	}

	loggerConfig.LogFile = "/tmp/log-test/test.log"
	if err := loggerConfig.globalClean(nil); err != nil {
		t.Error("Global clean failed:", err)
	}
	if _, err := os.Stat("/tmp/log-test/"); err != nil {
		t.Error("LoggerConfig.globalClean did not create dir:", err)
	}
	os.Remove("/tmp/log-test/")
}

func TestLoggerAppClean(t *testing.T) {
	config := &Config{
		Logger: &LoggerConfig{
			LogDir:      "/tmp/log-test/",
			MaxLogSize:  100,
			MaxLogsKept: -1,
			MaxLogAge:   -1,
		},
	}
	appConfig := &AppConfig{
		Name: "demo",
	}

	loggerConfig := &LoggerConfig{}
	if err := loggerConfig.appClean(config, appConfig); err != nil {
		t.Error("App clean failed:", err)
	}

	if loggerConfig.LogDir != config.Logger.LogDir {
		t.Error("LoggerConfig.LogDir not copied from global config")
	}
	if loggerConfig.MaxLogSize != config.Logger.MaxLogSize {
		t.Error("LoggerConfig.MaxLogSize not copied from global config")
	}
	if loggerConfig.MaxLogsKept != config.Logger.MaxLogsKept {
		t.Error("LoggerConfig.MaxLogsKept not copied from global config")
	}
	if loggerConfig.MaxLogAge != config.Logger.MaxLogAge {
		t.Error("LoggerConfig.MaxLogAge not copied from global config")
	}

	if loggerConfig.StdoutLogFile != "/tmp/log-test/app_demo.out" {
		t.Error("Incorrect default stdout log file set:", loggerConfig.StdoutLogFile)
	}
	if loggerConfig.StderrLogFile != "/tmp/log-test/app_demo.err" {
		t.Error("Incorrect default stderr log file set:", loggerConfig.StderrLogFile)
	}

	loggerConfig.StdoutLogFile = ""
	loggerConfig.StderrLogFile = ""
	loggerConfig.LogFile = "/tmp/log-test/demo.log"
	if err := loggerConfig.appClean(config, appConfig); err != nil {
		t.Error("App clean failed:", err)
	}

	if loggerConfig.StdoutLogFile != loggerConfig.LogFile {
		t.Error("StdoutLogFile should be set to LogFile.")
	}
	if loggerConfig.StderrLogFile != loggerConfig.LogFile {
		t.Error("StderrLogFile should be set to LogFile.")
	}

	if _, err := os.Stat("/tmp/log-test/"); err != nil {
		t.Error("LoggerConfig.appClean did not create dir:", err)
	}

	os.Remove("/tmp/log-test/")
}

func TestConfigClean(t *testing.T) {
	config := &Config{
		Logger: &LoggerConfig{
			LogDir: "/tmp/log-test/",
		},
		Apps: []*AppConfig{
			&AppConfig{
				Name:         "demo",
				Command:      "../demoapp/demoapp --port={port}",
				ExternalPort: 8000,
			},
			&AppConfig{
				Name:         "demo1",
				Command:      "../demoapp/demoapp --port={port}",
				ExternalPort: 8001,
			},
		},
	}
	if err := config.clean(nil); err != nil {
		t.Error("Config.clean fails with empty settings:", err)
	}

	if config.PortRange == nil {
		t.Error("Config.PortRange should be initialized")
	}
	if config.Rpc == nil {
		t.Error("config.Rpc should be initialized")
	}

	config.Apps = []*AppConfig{
		&AppConfig{
			Name:         "demo",
			Command:      "../demoapp/demoapp --port={port}",
			ExternalPort: 8000,
		},
		&AppConfig{
			Name:         "demo",
			Command:      "../demoapp/demoapp --port={port}",
			ExternalPort: 8001,
		},
	}
	if config.clean(nil) == nil {
		t.Error("Config.clean should fail with apps with same Name")
	}

	config.Apps = []*AppConfig{
		&AppConfig{
			Name:         "demo",
			Command:      "../demoapp/demoapp --port={port}",
			ExternalPort: 8000,
		},
		&AppConfig{
			Name:         "demo1",
			Command:      "../demoapp/demoapp --port={port}",
			ExternalPort: 8000,
		},
	}
	if config.clean(nil) == nil {
		t.Error("Config.clean should fail with apps with same ExternalPort")
	}
}

func TestConfigIncludeFile(t *testing.T) {
	config := &Config{}

	if config.includeFile("/not/a/path/gracevisor.yaml") != nil {
		t.Error("Config.includeFile should ignore any file named gracevisor.yaml")
	}

	if config.includeFile("/not/a/path/notconfig.txt") != nil {
		t.Error("Config.includeFile should ignore any file not .yaml")
	}

	if config.includeFile("/not/a/path/invalid_file.yaml") == nil {
		t.Error("Config.includeFile should fail with invalid file.")
	}

	if err := config.includeFile("../conf/app.yaml"); err != nil {
		t.Error("Error including '../conf/app.yaml':", err)
	}
	if len(config.Apps) != 1 {
		t.Error("App not added to Config.Apps")
	}
}

func TestConfigInclude(t *testing.T) {
	config := &Config{}

	if config.include("/not/a/path/") == nil {
		t.Error("Config.include should fail on folder that does not exist")
	}

	if config.include("/not/a/path/invalid_file.yaml") == nil {
		t.Error("Config.includeFile should fail on file that does not exist")
	}

	if err := config.include("../conf/"); err != nil {
		t.Error("Error including folder '../conf/':", err)
	}
	if len(config.Apps) != 1 {
		t.Error("App not added to Config.Apps")
	}

	config.Apps = []*AppConfig{}
	if err := config.include("../conf/app.yaml"); err != nil {
		t.Error("Error including file '../conf/app.yaml':", err)
	}
	if len(config.Apps) != 1 {
		t.Error("App not added to Config.Apps")
	}
}

func TestParseConfig(t *testing.T) {
	config, err := ParseConfing("/not/a/path")
	if err == nil {
		t.Error("Parsing of invalid path should fail.")
	}

	config, err = ParseConfing("../conf")

	if err != nil {
		t.Error("Parsing of sample config failed:", err)
	}

	if len(config.Apps) != 3 {
		t.Error("Sample config should load 3 apps.")
	}
}
