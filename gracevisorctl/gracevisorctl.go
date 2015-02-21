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

func mainT() {
	client, err := rpc.DialHTTP("tcp", "localhost:1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}

	var reply string
	err = client.Call("Rpc.Restart", "demo1", &reply)
	if err != nil {
		log.Fatal("Restart error:", err)
	}
	fmt.Println("Restart in progress")

	err = client.Call("Rpc.Status", "", &reply)
	if err != nil {
		log.Fatal("Status error:", err)
	}
	fmt.Println(reply)
}

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
	}

	app.Run(os.Args)
}
