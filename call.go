package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
)

// call            host:port command arg arg .. arg
var callCommand = cli.Command{
	Name:        "call",
	Usage:       "run command in redis cluster.",
	ArgsUsage:   `host:port command arg arg .. arg`,
	Description: `The call command for call cmd in every redis cluster node.`,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "password, a",
			Value: "",
			Usage: `password, the default value is ""`,
		},
	},
	Action: func(context *cli.Context) error {
		if context.NArg() < 2 {
			fmt.Printf("Incorrect Usage.\n\n")
			cli.ShowCommandHelp(context, "call")
			logrus.Fatalf("Must provide \"host:port command\" for call command!")
		}

		if context.String("password") != "" {
			RedisPassword = context.String("password")
		}

		rt := NewRedisTrib()
		if err := rt.CallClusterCmd(context); err != nil {
			return err
		}
		return nil
	},
}

func (rt *RedisTrib) CallClusterCmd(context *cli.Context) error {
	var addr string

	if addr = context.Args().Get(0); addr == "" {
		return errors.New("please check host:port for call command")
	}

	if err := rt.LoadClusterInfoFromNode(addr); err != nil {
		return err
	}

	cmd := strings.ToUpper(context.Args().Get(1))
	cmdArgs := ToInterfaceArray(context.Args()[2:])

	if context.String("password") != "" {
		RedisPassword = context.String("password")
	}

	logrus.Printf(">>> Calling %s %s", cmd, cmdArgs)
	_, err := rt.EachRunCommandAndPrint(cmd, cmdArgs...)
	if err != nil {
		logrus.Errorf("Command failed: %s", err)
		return err
	}

	return nil
}
