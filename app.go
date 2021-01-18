package main

import (
	"os"
	"strings"
	"zhub/cli"
	"zhub/conf"
	"zhub/zsub"
)

func main() {
	server := true
	confPath := "app.conf"
	addr := ""

	for _, arg := range os.Args[1:] {
		if strings.EqualFold(arg, "cli") {
			server = false
		} else if strings.Index(arg, "-d=") == 0 {
			addr = arg[3:]
		} else if strings.Index(arg, "-c=") == 0 {
			confPath = arg[3:]
		}
	}

	if server {
		conf.Load(confPath)
		if len(addr) == 0 {
			addr = conf.GetStr("service.zhub.servers", "127.0.0.1:1216")
		}
		// 服务进程启动
		zsub.ServerStart(addr)
	} else {
		cli.ClientRun(addr)
	}

}
