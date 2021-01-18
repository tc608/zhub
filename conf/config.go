package conf

import (
	"bufio"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

var (
	config   = make(map[string]string)
	LogDebug bool
)

func Load(path string) {
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
