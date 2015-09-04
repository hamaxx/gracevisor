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

Nightly binary packages (only supports upstart init service at the moment)

- [deb x86_64](https://s3.amazonaws.com/gracevisor/gracevisor_nightly_amd64.deb)
- [rpm x86_64](https://s3.amazonaws.com/gracevisor/gracevisor-nightly-1.x86_64.rpm)
- [tar.gz x86_64](https://s3.amazonaws.com/gracevisor/gracevisor_nightly_x86_64.tar.gz)

Build and install gracevisor

    go get github.com/hamaxx/gracevisor/{gracevisord,gracevisorctl}

Put your gracevisor.yaml file into /etc/gracevisor or pass the config dir as a paramater

    ./gracevisord --conf ./conf

Run gracevisorctl to see the options

    ./gracevisorctl -h

## Configuration for gracevisord

By default configuration is located in */etc/gracevisor/gracevisor.yaml*, but can be changed by passing the config dir as a parameter:

	./gracevisord --conf ./conf

The configuration format is [yaml](http://www.yaml.org/spec/1.2/spec.html).
All configuration options are optional, except for app **name** and **command**.

### Example:
```yaml
port_range:
    from: 10000
    to: 11000

rpc:
    host: localhost
    port: 9001

logger:
    log_dir: "../log"
    max_log_size: 500
    max_logs_kept: -1
    max_log_age: -1

user:
    username: root

apps_include: ["../conf"]

apps:
    - name: demo
      command: ../demoapp/demoapp --port={port}
```

### port_range:
port_range specifies the range of ports for internal use for applications. Default range is from *10000* to *11000*.

Options:
- **from:** Lower bound of port range.
- **to:** Upper bound of port range.

### rpc:
rpc specifies options for rpc server.

Options:
- **host:** Rpc server hostname. Default is *localhost*.
- **port:** Rpc server port. Default is *9001*.

### logger:
logger specifies global logger settings.

Options:

- **log_dir:** Directory for logs. Log for gracevisor *gracevisor.log* will be stored here if not overriden with **log_file**. This option will be inherited in apps if not overridden. Default is */var/log/gracevisor*.

- **log_file:** Log file for gracevisor. Default is *gracevisor.log* in **log_dir** folder. This is a path relative to gracevisord working dir not **log_dir**.

- **max_log_size:** Max size for log files (in megabytes). This option will be inherited in apps if not overridden. Default is *500*.

- **max_logs_kept:** Maximum number of logs to be kept after log rotation. This option will be inherited in apps if not overridden. Default is to keep all logs.

- **max_log_age:** Maximum log age before rotating it. This option will be inherited in apps if not overridden. Default is no age limit.

### user:
user is a global option for user under which to run apps. This option wil be inherited in apps and can be overriden there. If no user is specified, the app will be run with the same user as *gracevisord*.

Options:
- **username:** Name of the user.

### apps_include:

apps_include specifies additional configuration files for apps. Each file has to be a valid yaml file for one app (see **Application** for options). This option takes a list of paths that can be either folders of yaml files or specific yaml files.

### apps:

apps specifies a list of configurations for apps, see **Application** for valid options.

### Application

Application is a configuration for an app, that can be specified in separate file using **apps_include** or in main configuration file using **apps**.

Options:

- **name**: (required) Name to identify the app.

- **command**: (required) Command to execute the app. Either this option or **environment** has to include *{port}* badge, that will be used to specify the internal port on which the app should run.

- **environment**: A list of environment variables to set for the app. Format for this option is a list of strings. Example: *["PORT={port}"]*

- **directory**: Working directory in which the app should be run.

- **healthcheck**: Http path for the app that should return 200 as long as app is working correctly, otherwise the app will be restarted.

- **internal_host**: Internal host on which app can be accessed. Default is *localhost*.

- **external_host**: External host on which the app should listen. Default is *localhost*.

- **external_port**: External port for the app. Default is *8080*.

- **proxy:** Type of proxy. Options are *tcp* and *http*. Default is *http*.

- **stop_signal**: Signal to be used to shutdown running app. Default is *TERM*.

- **max_retries**: Maximum number of retries to start the app. Default is *5*.

- **start_timeout**: Timeout to wait for app to start before retrying. Default is no timeout.

- **stop_timeout**: Timeout to wait for app to exit after sending **stop_signal** before killing it. Default is no timeout.

- **user**: User under which the app should run. If not specified, the option will be inherited from global setting. If nothing is specified, the app will run with the same user as *gracevisord*.
Options:
  - **username**: Name of the user.

- **logger**: Settings for logging *stdout* and *stderr* for app.
Options:

  - **log_dir**: Directory for logs. If not specified this option will be inherited from global logger config. If no other option is specified, log files will be in this folder: *app_{appname}.out* and *app_{appname}.err*.

  - **log_file**: Log file for both *stdout* and *stderr*. This is a path relative to gracevisord working dir not **log_dir**. They will be logged to the same file, except if one is overridden with following options.

  - **stdout_log_file**: Log file for *stdout*. This is a path relative to gracevisord working dir not **log_dir**.

  - **stderr_log_file**: Log file for *stderr*. This is a path relative to gracevisord working dir not **log_dir**.

  - **max_log_size:** Max size for log files (in megabytes). If not specified this option will be inherited from global logger config.

  - **max_logs_kept:** Maximum number of logs to be kept after log rotation. If not specified this option will be inherited from global logger config.

  - **max_log_age:** Maximum log age before rotating it. If not specified this option will be inherited from global logger config.


## TODO

- apps management: reload config, remove, add
- init scripts for systemd and init.d
- docs
- **tests**
- ...

**Long term**

- alerts
- statsd supports
- web interface
- fg command
