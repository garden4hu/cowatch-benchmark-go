package main

import (
	"fmt"
	"github.com/fatih/color"
)

func InstanceLoading(conf *Config) {
	if err := getRooms(conf); err != nil {
		color.Set(color.FgRed)
		fmt.Println(err)
		color.Unset()
		return
	}
	if err := getUsers(conf); err != nil {
		color.Set(color.FgRed)
		fmt.Println(err)
		color.Unset()
		return
	}
}
