package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"zhub/cli"
	"zhub/zdb"
)

func main() {
	server := true
	host := "127.0.0.1"
	port := 1216

	for _, arg := range os.Args[1:] {
		if strings.EqualFold(arg, "cli") {
			server = false
		} else if strings.Index(arg, "-h=") == 0 {
			host = arg[3:]
		} else if strings.Index(arg, "-p=") == 0 {
			p, err := strconv.Atoi(arg[3:])
			if err != nil {
				fmt.Println("-Error para: -p=[number]")
				os.Exit(0)
			}
			port = p
		}
	}

	if server {
		zdb.ServerStart(host, port)
	} else {
		cli.ClientRun(host, port)
	}

}
