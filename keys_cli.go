package main

import (
	"fmt"
	"github.com/dynport/gocli"
	"os"
	"strconv"
)

func init() {
	cli.Register(
		"key/list",
		&gocli.Action{
			Description: "List available ssh keys",
			Handler:     ListKeys,
		},
	)
}

func ListKeys(args *gocli.Args) error {
	table := gocli.NewTable()
	table.Add("Id", "Name")
	keys, e := CurrentAccount().SshKeys()
	if e != nil {
		return e
	}
	for _, key := range keys {
		table.Add(strconv.Itoa(key.Id), key.Name)
	}
	fmt.Fprintln(os.Stdout, table.String())
	return nil
}
