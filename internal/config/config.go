package config

import (
	"github.com/spf13/viper"
	"log"
	"os"
)

type Log struct {
	Handlers string
	Level    string
	File     string
}

type Config struct {
	Log     Log
	Service struct {
		Watch string
		Addr  string
		Auth  bool
	}
	Data struct {
		Dir string
	}
	Ztimer struct {
		Db struct {
			Addr     string
			User     string
			Password string
			Database string
		}
	}
	Auth map[string]string
}

func ReadConfig() Config {
	conf := Config{}
	viper.SetDefault("log.handlers", "console")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("service.auth", true)

	/*// 读取指定的配置文件
	if !strings.EqualFold("", fileName) {
		viper.AddConfigPath(fileName) // 指定配置文件
		if err := viper.ReadInConfig(); err == nil {
			if err := viper.Unmarshal(&conf); err != nil {
				log.Fatalf("Failed to unmarshal config: %s", err.Error())
			}
			return conf
		}

		log.Fatalf("Config file not found: " + fileName)
		return conf
	}*/

	// 尝试从 /etc/ 目录下查找 zhub.ini 配置文件
	viper.AddConfigPath("/etc/") // 添加 /etc/ 目录作为配置文件搜索路径
	viper.SetConfigName("zhub")  // 指定配置文件名为 zhub
	if err := viper.ReadInConfig(); err == nil {
		if err := viper.Unmarshal(&conf); err != nil {
			log.Fatalf("Failed to unmarshal config: %s", err.Error())
		}
		return conf
	}
	// 如果 /etc/ 目录下未找到配置文件，则尝试从当前程序运行目录下查找 app.ini 配置文件
	dir, err := os.Getwd() // 获取程序运行目录
	if err != nil {
		log.Fatalf("Failed to get current directory: %s", err.Error())
	}
	viper.SetConfigName("app") // 指定配置文件名为 app
	viper.SetConfigType("ini") // 指定配置文件类型为 ini
	viper.AddConfigPath(dir)   // 添加当前程序所在目录作为配置文件搜索路径
	if err := viper.ReadInConfig(); err == nil {
		if err := viper.Unmarshal(&conf); err != nil {
			log.Fatalf("Failed to unmarshal config: %s", err.Error())
		}
		return conf
	}
	// 如果在 /etc/ 目录和当前程序所在目录下均未找到配置文件，则报错
	log.Fatalf("Config file not found")
	return conf
}
func InitLog(logConfig Log) {
	logHandlers := logConfig.Handlers
	logLevel := logConfig.Level
	logFile := logConfig.File

	if logHandlers == "console" {
		log.SetOutput(os.Stdout)
	} else if logHandlers == "file" {
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_SYNC|os.O_RDWR, 0777)
		if err != nil {
			log.Println(err)
		}
		log.SetOutput(file)
	} else {
		log.SetOutput(os.Stdout)
	}

	switch logLevel {
	case "info":
		log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
		log.SetPrefix("[Info] ")
		log.Println("Logger is set up with log level: info")
	case "debug":
		log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
		log.SetPrefix("[Debug] ")
		log.Println("Logger is set up with log level: debug")
	case "error":
		log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
		log.SetPrefix("[Error] ")
		log.Println("Logger is set up with log level: error")
	default:
		log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
		log.SetPrefix("[Info] ")
		log.Println("Logger is set up with default log level: info")
	}
}
