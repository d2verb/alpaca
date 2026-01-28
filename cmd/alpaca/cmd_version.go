package main

import "fmt"

type VersionCmd struct{}

func (c *VersionCmd) Run() error {
	fmt.Printf("alpaca version %s\n", version)
	return nil
}
