# gracevisor
A Process Control System Built for the Web

## Goal

Goal of the project is to provide an all in one general solution for process supervision with graceful/hitless application reloads.

## Motivation

We can solve graceful restarts on two levels. Either in application or in infrastructure. Each has it's own problems and benefits. Graceful application restarts are always language/framework specific and are often incompatible with process supervision solutions like supervisor. Solving zero package loss restarts in infrastructure is general but often much more complicated since we have to add a communication layer between applications and infrastructure. This usually involves an api access to a load balancers and more complicated process supervision since we have to handle multiple live instances(new and old) of applications.

Gracevisor is trying to take the infrastructure approach and package it in an easy to understand and manage solution. To achieve that we merged a load balancer(reverse proxy) and a process supervisor into a single process where communication is not a problem.

## Overview

**Restart process:**

- Start a new instance of the application on an empty port
- Wait for the application to start
- Switch traffic from the old instance to the newly created one
- When all requests are processed, send a stop signal to the old instance

## Progress

At the moment we're building a proof of concept. It's not ready for production yet and it won't be any time soon. But if you want to contribute with ideas or code, you're very welcome. Open an issue or send me an email.

## I know it doesn't work yet, but I still want to try it out

Build and install gracevisor

    go get github.com/hamaxx/gracevisor/{gracevisord,gracevisorctl}

Put your gracevisor.yaml file into /etc/gracevisor or pass the config dir as a paramater

    ./gracevisord --conf ./conf

Run gracevisorctl to see the options

    ./gracevisorctl -h

## TODO

- configure env
- config: validation, default values, include statement
- apps management: reload config, remove, add
- daemonize gracevisord (maybe it would be easier to just write systemd/upstart/... scripts)
- install script
- docs
- **tests**
- ...

**Long term**

- alerts
- statsd supports
- web interface
- fg command
