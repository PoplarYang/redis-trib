package main

import (
	"errors"
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
)

// fix            host:port
//                  --timeout <arg>
var fixCommand = cli.Command{
	Name:        "fix",
	Usage:       "fix the redis cluster.",
	ArgsUsage:   `host:port`,
	Description: `The fix command for fix the redis cluster.`,
	Flags: []cli.Flag{
		cli.IntFlag{
			Name:  "timeout, t",
			Value: MigrateDefaultTimeout,
			Usage: `timeout for fix the redis cluster.`,
		},
		cli.StringFlag{
			Name:  "password, a",
			Value: "",
			Usage: `password, the default value is "".`,
		},
		cli.StringFlag{
			Name:  "password, a",
			Value: "",
			Usage: `password, the default value is ""`,
		},
	},
	Action: func(context *cli.Context) error {
		if context.NArg() != 1 {
			fmt.Printf("Incorrect Usage.\n\n")
			cli.ShowCommandHelp(context, "fix")
			logrus.Fatalf("Must provide at least \"host:port\" for fix command!")
		}

		if context.String("password") != "" {
			RedisPassword = context.String("password")
		}

		rt := NewRedisTrib()
		if err := rt.FixClusterCmd(context); err != nil {
			return err
		}
		return nil
	},
}

func (rt *RedisTrib) FixClusterCmd(context *cli.Context) error {
	var addr string

	if addr = context.Args().Get(0); addr == "" {
		return errors.New("please check host:port for fix command")
	}

	rt.SetFix(true)
	timeout := context.Int("timeout")
	rt.SetTimeout(timeout)
	if err := rt.LoadClusterInfoFromNode(addr); err != nil {
		return err
	}

	rt.CheckCluster(false)
	return nil
}
