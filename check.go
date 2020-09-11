package main

import (
	"errors"
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
)

// check            host:port
var checkCommand = cli.Command{
	Name:        "check",
	Usage:       "check the redis cluster.",
	ArgsUsage:   `host:port`,
	Description: `The check command check for redis cluster.`,
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
			cli.ShowCommandHelp(context, "check")
			logrus.Fatalf("Must provide host:port for check command!")
		}

		if context.String("password") != "" {
			RedisPassword = context.String("password")
		}

		rt := NewRedisTrib()
		if err := rt.CheckClusterCmd(context); err != nil {
			return err
		}
		return nil
	},
}

func (rt *RedisTrib) CheckClusterCmd(context *cli.Context) error {
	var addr string

	if addr = context.Args().Get(0); addr == "" {
		return errors.New("please check host:port for check command")
	}

	if err := rt.LoadClusterInfoFromNode(addr); err != nil {
		return err
	}

	rt.CheckCluster(false)
	return nil
}
