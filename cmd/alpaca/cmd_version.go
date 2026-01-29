package main

import "fmt"

type VersionCmd struct{}

func (c *VersionCmd) Run() error {
	fmt.Printf("alpaca version %s (%s)\n", version, commit)
	return nil
}
