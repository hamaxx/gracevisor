package main

import (
	"fmt"
	"log"
	"net/rpc"
	"os"
	"text/tabwriter"
	"time"

	"github.com/hamaxx/gracevisor/common/report"
	"github.com/hamaxx/gracevisor/deps/cli"
)

const (
	defaultHost = "localhost"
	defaultPort = 9001
)

func getRpcClient(c *cli.Context) *rpc.Client {
	client, err := rpc.DialHTTP("tcp", fmt.Sprintf("%s:%d", c.GlobalString("host"), c.GlobalInt("port")))
	if err != nil {
		log.Fatal("dialing:", err)
	}
	return client
}

func basicRpcCall(client *rpc.Client, method string, args interface{}) {
	var reply string
	err := client.Call(fmt.Sprintf("Rpc.%s", method), args, &reply)
	if err != nil {
		log.Fatal("error:", err)
	}
	if reply != "" {
		fmt.Println(reply)
	}
}

func statusRpcCall(client *rpc.Client, args interface{}) {
	var reply []*report.App
	err := client.Call("Rpc.Status", args, &reply)
	if err != nil {
		log.Fatal("error:", err)
	}

	tabWriter := tabwriter.NewWriter(os.Stdout, 2, 2, 1, ' ', 0)
	for _, appReport := range reply {
		fmt.Fprintf(tabWriter, "[%s/%s:%d]\n", appReport.Name, appReport.Host, appReport.Port)

		for _, instanceReport := range appReport.Instances {
			if instanceReport.Active {
				fmt.Fprint(tabWriter, "*\t")
			} else {
				fmt.Fprint(tabWriter, "\t")
			}

			fmt.Fprintf(tabWriter, "%d/%s:%d\t", instanceReport.Id, instanceReport.Host, instanceReport.Port)

			fmt.Fprintf(tabWriter, "%s\t", instanceReport.Status)

			fmt.Fprintf(tabWriter, "%s\t", time.Duration(instanceReport.SinceStatusChange)*time.Second)

			fmt.Fprintf(tabWriter, "%s\n", instanceReport.Error)
		}
	}

	tabWriter.Flush()
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
				statusRpcCall(getRpcClient(c), c.Args().First())
			},
		},
		{
			Name:  "restart",
			Usage: "restart application",
			Action: func(c *cli.Context) {
				basicRpcCall(getRpcClient(c), "Restart", c.Args().First())
			},
		},
		{
			Name:  "start",
			Usage: "start application",
			Action: func(c *cli.Context) {
				basicRpcCall(getRpcClient(c), "Start", c.Args().First())
			},
		},
		{
			Name:  "stop",
			Usage: "stop running instances",
			Action: func(c *cli.Context) {
				basicRpcCall(getRpcClient(c), "Stop", c.Args().First())
			},
		},
		{
			Name:  "kill",
			Usage: "kill running instances",
			Action: func(c *cli.Context) {
				basicRpcCall(getRpcClient(c), "Kill", c.Args().First())
			},
		},
	}

	app.Run(os.Args)
}
