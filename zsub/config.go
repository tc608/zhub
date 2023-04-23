package zsub

import (
	"bufio"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

var (
	dir, _   = os.Getwd()
	config   = make(map[string]string)
	LogDebug bool
	datadir  = dir + "/data"
)

func LoadConf(path string) {
	//log.Println("APP_CONF =", path)
	f, err := os.Open(path)
	if err != nil {
		log.Panicln(err)
	}

	reader := bufio.NewReader(f)
	space := ""
	for {
		bytes, err := reader.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		line := string(bytes)
		line = strings.Trim(line, " \r\n")
		if len(line) == 0 {
			continue
		}
		if strings.Contains(line, "#") {
			line = line[0:strings.Index(line, "#")]
		}

		switch {
		case strings.EqualFold(line, ""):
		case strings.Index(line, "[") == 0 && strings.Index(line, "]") > 0:
			space = line[1:strings.Index(line, "]")]
			space = strings.Trim(space, " ")
		case strings.Index(line, "=") > 0:
			arr := strings.Split(line, "=")
			if len(arr) < 2 {
				continue
			}

			config[space+"."+strings.Trim(arr[0], " ")] = strings.Trim(arr[1], " ")
		default:
			continue
		}
	}

	LogDebug = strings.EqualFold(config["log.level"], "debug")

	datadir = GetStr("data.dir", "${APP_HOME}/data")
	datadir = strings.ReplaceAll(datadir, "${APP_HOME}", dir)

	os.MkdirAll(datadir, os.ModeDir)
	os.Chmod(datadir, 0777)

	initLog()
}

func GetStr(key string, def string) string {
	if len(config[key]) == 0 {
		return def
	}
	return config[key]
}

func GetInt(key string, def int) int {
	if len(config[key]) == 0 {
		return def
	}
	n, err := strconv.Atoi(config[key])
	if err != nil {
		log.Println(err, "return def;")
		return def
	}
	return n
}

func initLog() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("initLog Err:", r)
		}
	}()

	file, err := os.OpenFile("zhub.log", os.O_CREATE|os.O_APPEND|os.O_SYNC|os.O_RDWR, 0777)
	if err != nil {
		log.Println(err)
	}
	log.SetOutput(file)

	/*
		if strings.EqualFold(GetStr("log.handlers", "console"), "console") {
			return
		}

		var logfile = GetStr("log.pattern", "${APP_HOME}/logs-200601/log-20060102.log")

		c := cron.New()
		fun := func() {
			now := time.Now()
			logfile := strings.ReplaceAll(logfile, "${APP_HOME}", dir)
			logfile = now.Format(logfile)

			if strings.LastIndexAny(logfile, "/") > 0 {
				logdir := logfile[0:strings.LastIndexAny(logfile, "/")]
				os.MkdirAll(logdir, 0666)
			}

			file, err := os.OpenFile(logfile, os.O_CREATE|os.O_APPEND|os.O_SYNC|os.O_RDWR, 0777)
			if err != nil {
				log.Println(err)
			}

			//log.Println("SET LOG_FILE =", file.Name())
			log.SetOutput(file)
		}
		fun()

		c.AddFunc("0 0 * * * *", fun)
		go c.Run()
	*/
}
