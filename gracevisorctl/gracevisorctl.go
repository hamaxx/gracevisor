package main

import (
	"fmt"
	"log"
	"net/rpc"
	"os"

	"github.com/codegangsta/cli"
)

const (
	defaultHost = "localhost"
	defaultPort = 9001
)

func basicRpcCall(c *cli.Context, method string, args interface{}) {
	client, err := rpc.DialHTTP("tcp", fmt.Sprintf("%s:%d", c.GlobalString("host"), c.GlobalInt("port")))
	if err != nil {
		log.Fatal("dialing:", err)
	}

	var reply string
	err = client.Call(fmt.Sprintf("Rpc.%s", method), args, &reply)
	if err != nil {
		log.Fatal("error:", err)
	}
	if reply != "" {
		fmt.Println(reply)
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "gracevisorctl"
	app.Usage = "Manage gracevisord"
	app.Email = "jure@hamsworld.net"
	app.Version = "0.0.1"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "host",
			Value: defaultHost,
			Usage: "daemon host",
		},
		cli.IntFlag{
			Name:  "port",
			Value: defaultPort,
			Usage: "daemon port",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:  "status",
			Usage: "display application status",
			Action: func(c *cli.Context) {
				basicRpcCall(c, "Status", "")
			},
		},
		{
			Name:  "restart",
			Usage: "restart application",
			Action: func(c *cli.Context) {
				basicRpcCall(c, "Restart", c.Args().First())
			},
		},
		{
			Name:  "start",
			Usage: "start application",
			Action: func(c *cli.Context) {
				basicRpcCall(c, "Start", c.Args().First())
			},
		},
		{
			Name:  "stop",
			Usage: "stop running instances",
			Action: func(c *cli.Context) {
				basicRpcCall(c, "Stop", c.Args().First())
			},
		},
		{
			Name:  "kill",
			Usage: "kill running instances",
			Action: func(c *cli.Context) {
				basicRpcCall(c, "Kill", c.Args().First())
			},
		},
	}

	app.Run(os.Args)
}
