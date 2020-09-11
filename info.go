package main

import (
	"errors"
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
)

// info            host:port
var infoCommand = cli.Command{
	Name:        "info",
	Usage:       "display the info of redis cluster.",
	ArgsUsage:   `host:port`,
	Description: `The info command get info from redis cluster.`,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "password, a",
			Value: "",
			Usage: `password, the default value is ""`,
		},
	},
	Action: func(context *cli.Context) error {
		if context.NArg() != 1 {
			fmt.Printf("Incorrect Usage.\n\n")
			cli.ShowCommandHelp(context, "info")
			logrus.Fatalf("Must provide host:port for info command!")
		}

		if context.String("password") != "" {
			RedisPassword = context.String("password")
		}

		rt := NewRedisTrib()
		if err := rt.InfoClusterCmd(context); err != nil {
			return err
		}
		return nil
	},
}

func (rt *RedisTrib) InfoClusterCmd(context *cli.Context) error {
	var addr string

	if addr = context.Args().Get(0); addr == "" {
		return errors.New("please check host:port for info command")
	}

	if err := rt.LoadClusterInfoFromNode(addr); err != nil {
		return err
	}

	rt.ShowClusterInfo()
	return nil
}
