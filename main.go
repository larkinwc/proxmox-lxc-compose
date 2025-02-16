package main

import (
	"log"

	"github.com/larkinwc/proxmox-lxc-compose/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		log.Fatal(err)
	}
}
