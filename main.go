package main

import (
	"log"
	"os"
	"strings"
	"time"
	"zhub/cmd"
	"zhub/zsub"
)

var (
	dir, _   = os.Getwd()
	confPath = dir + "/app.ini" // 配置文件地址
	server   = true
	addr     = "" // 服务地址
)

func main() {
	for _, arg := range os.Args[1:] {
		if strings.EqualFold(arg, "cli") {
			server = false
		} else if strings.Index(arg, "-d=") == 0 {
			addr = arg[3:]
		} else if strings.Index(arg, "-c=") == 0 {
			confPath = arg[3:]
		}
	}
	zsub.LoadConf(confPath)
	if len(addr) == 0 {
		addr = zsub.GetStr("service.zhub.servers", "127.0.0.1:1216")
	}

	if len(os.Args) == 3 && strings.EqualFold(os.Args[1], "-r") {
		if cli, err := cmd.Create("zhub-local", addr, "group-admin", "zchd@123456"); err != nil {
			log.Println(err)
		} else {
			switch os.Args[2] {
			case "timer":
				cli.Cmd("reload-timer")
			case "shutdown", "stop":
				cli.Cmd("shutdown")
			}
			cli.Close()
			time.Sleep(time.Millisecond * 10)
		}
		return
	}

	if server {
		go zsub.StartWatch()
		zsub.StartServer(addr) // 服务进程启动
	} else {
		cmd.ClientRun(addr)
	}

}
