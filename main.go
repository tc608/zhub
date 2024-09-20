package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"zhub/cmd"
	"zhub/internal/config"
	"zhub/internal/monitor"
	"zhub/internal/zbus"
)

func main() {
	// 命令查询版本号
	versionFlag := flag.Bool("version", false, "Display the version")
	vFlag := flag.Bool("v", false, "Display the version")
	VFlag := flag.Bool("V", false, "Display the version")

	isCliMode := flag.Bool("cli", false, "Run as client mode") // 客户端模式参数
	rcmd := flag.String("r", "", "Run command in client mode") // 客户端命令参数

	// 解析命令行参数
	flag.Parse()

	// 检查是否有版本参数, 如果有则输出版本号并退出
	if *versionFlag || *vFlag || *VFlag {
		fmt.Printf("Version: %s\n", monitor.Version)
		os.Exit(0) // 输出后退出
	}

	conf := config.ReadConfig() // 读取配置文件
	addr := conf.Service.Addr   // 获取服务地址
	config.InitLog(conf.Log)    // 初始化日志配置
	// 输出版本号
	log.Println("ZHub version:", monitor.Version)

	{
		/*
			使用环境变量覆盖 配置文件参数 TODO
			port, err := strconv.Atoi(os.Getenv("PORT"))
			if err != nil {
				port = 6066
			}*/
	}

	if *rcmd != "" { // 如果指定了客户端命令
		adminToken, err := zbus.AuthManager.AdminToken() // 认证信息
		if err != nil {
			log.Fatal(err) // Configuration error, stop the client from running.
			return
		}

		cli := cmd.ZHubClient{}
		err = cli.Initx("server-local", addr, "server-admin", adminToken)

		// cli, err := cmd.Create("server-local", addr, "server-admin", adminToken) // 创建客户端连接
		if err != nil {
			log.Println(err) // 如果连接失败则打印错误信息
			return
		}
		defer cli.Close() // 延迟关闭客户端连接
		switch *rcmd {
		case "timer":
			cli.Cmd("reload-timer")
		case "shutdown", "stop":
			cli.Cmd("shutdown")
		}
		return
	}
	if *isCliMode {
		cmd.ClientRun(addr) // 客户端运行
	} else {
		go monitor.StartWatch()      // 启动监控协程
		zbus.StartServer(addr, conf) // 启动服务进程
	}
}
