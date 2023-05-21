package main

import (
	"flag"
	"log"
	"zhub/cmd"
	"zhub/internal/config"
	"zhub/zsub"
)

func main() {
	var isCliMode bool                                           // 是否以客户端模式运行的标志
	var rcmd string                                              // 客户端模式下运行的命令
	flag.BoolVar(&isCliMode, "cli", false, "run as client mode") // 定义 cli 参数
	flag.StringVar(&rcmd, "r", "", "run as client mode")         // 定义 r 参数
	flag.Parse()                                                 // 解析命令行参数

	conf := config.ReadConfig() // 读取配置文件
	addr := conf.Service.Addr   // 获取服务地址
	config.InitLog(conf.Log)    // 初始化日志配置

	if rcmd != "" { // 如果指定了客户端命令
		auth := ""                          // 认证信息
		for key, value := range conf.Auth { // 遍历找到一个认证信息
			auth = key + "@" + value
			break
		}
		cli, err := cmd.Create("zhub-local", addr, "group-admin", auth) // 创建客户端连接
		if err != nil {
			log.Println(err) // 如果连接失败则打印错误信息
			return
		}
		defer cli.Close() // 延迟关闭客户端连接
		switch rcmd {
		case "timer":
			cli.Cmd("reload-timer")
		case "shutdown", "stop":
			cli.Cmd("shutdown")
		}
		return
	}
	if isCliMode {
		cmd.ClientRun(addr) // 客户端运行
	} else {
		go zsub.StartWatch()         // 启动监控协程
		zsub.StartServer(addr, conf) // 启动服务进程
	}
}
