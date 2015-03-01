package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/hamaxx/gracevisor/deps/lumberjack"
)

var logLinePool = sync.Pool{}

type LogLine struct {
	line       bytes.Buffer
	time       time.Time
	instanceId uint32
}

func (ll *LogLine) String() string {
	return fmt.Sprintf("[%d/%s] %s", ll.instanceId, ll.time, ll.line.String())
}

func (ll *LogLine) WriteTo(w io.Writer) error {
	//TODO: no garbage
	if _, err := w.Write([]byte(ll.String())); err != nil {
		return err
	}
	if _, err := w.Write([]byte("\n")); err != nil {
		return err
	}
	return nil
}

type AppLogger struct {
	app *App

	stdoutWriter io.WriteCloser
	stderrWriter io.WriteCloser
}

func NewAppLogger(app *App) *AppLogger {
	stdoutWriter := &lumberjack.Logger{
		Filename:   app.config.StdoutLogFile,
		MaxSize:    app.loggerConfig.MaxLogSize,
		MaxAge:     app.loggerConfig.MaxLogAge,
		MaxBackups: app.loggerConfig.MaxLogsKept,
	}

	stderrWriter := &lumberjack.Logger{
		Filename:   app.config.StderrLogFile,
		MaxSize:    app.loggerConfig.MaxLogSize,
		MaxAge:     app.loggerConfig.MaxLogAge,
		MaxBackups: app.loggerConfig.MaxLogsKept,
	}

	return &AppLogger{
		app:          app,
		stdoutWriter: stdoutWriter,
		stderrWriter: stderrWriter,
	}
}

func (al *AppLogger) logStdout(logLine *LogLine) {
	if err := logLine.WriteTo(al.stdoutWriter); err != nil {
		log.Print("Stdout write error:", err)
	}
	logLinePool.Put(logLine)
}

func (al *AppLogger) logStderr(logLine *LogLine) {
	if err := logLine.WriteTo(al.stderrWriter); err != nil {
		log.Print("Stderr write error:", err)
	}

	logLinePool.Put(logLine)
}

type InstanceLogger struct {
	instance *Instance
}

func NewInstanceLogger(instance *Instance) (*InstanceLogger, error) {
	il := &InstanceLogger{
		instance: instance,
	}

	stdout, err := instance.exec.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := instance.exec.StderrPipe()
	if err != nil {
		return nil, err
	}

	il.lineReader(stdout, instance.app.appLogger.logStdout)
	il.lineReader(stderr, instance.app.appLogger.logStderr)

	return il, nil
}

func (il *InstanceLogger) lineReader(pipe io.ReadCloser, writer func(*LogLine)) {
	rd := bufio.NewReader(pipe)
	go func() {
		for {
			line, err := rd.ReadBytes('\n')
			if err == io.EOF {
				return
			} else if err != nil {
				log.Print("Read Error:", err)
				return
			}
			if len(line) > 0 && line[len(line)-1] == '\n' {
				line = line[0 : len(line)-1]
			}
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[0 : len(line)-1]
			}
			ll, err := il.newLogLine(line)
			if err != nil {
				log.Print("Log write error:", err)
				continue
			}
			writer(ll)
		}
	}()
}

func (il *InstanceLogger) newLogLine(line []byte) (*LogLine, error) {
	var logLine *LogLine

	if v := logLinePool.Get(); v != nil {
		logLine = v.(*LogLine)
		logLine.line.Reset()
	} else {
		logLine = &LogLine{}
	}

	if _, err := logLine.line.Write(line); err != nil {
		return nil, err
	}
	logLine.time = time.Now()
	logLine.instanceId = il.instance.id

	return logLine, nil
}
