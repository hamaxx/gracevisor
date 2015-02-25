package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"sync"
	"time"
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

type AppLogger struct {
	app *App
}

func NewAppLogger(app *App) *AppLogger {
	return &AppLogger{
		app: app,
	}
}

func (al *AppLogger) logStdout(logLine *LogLine) {
	// TODO log to file
	fmt.Println("out", al.app.config.Name, logLine)
	logLinePool.Put(logLine)
}

func (al *AppLogger) logStderr(logLine *LogLine) {
	// TODO log to file
	fmt.Println("err", al.app.config.Name, logLine)
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
			writer(il.newLogLine(line))
		}
	}()
}

func (il *InstanceLogger) newLogLine(line []byte) *LogLine {
	var logLine *LogLine

	if v := logLinePool.Get(); v != nil {
		logLine = v.(*LogLine)
		logLine.line.Reset()
	} else {
		logLine = &LogLine{}
	}

	logLine.line.Write(line)
	logLine.time = time.Now()
	logLine.instanceId = il.instance.id

	return logLine
}
