package zsub

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
// hubChan = make(chan Message, 1000) //接收到的 所有消息数据
)

// 数据封装
type Message struct {
	Conn *ZConn
	Rcmd []string
}

// 文件追加内容
func Append(str string, fileName string) {
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()

	_, err = file.WriteString(str)
	if err != nil {
		log.Println(err)
	}
}

// 数据持久化
func (s *ZSub) saveDelay() {
	s.Lock()
	defer s.Unlock()
	err := os.Remove("delay.z")
	if err != nil {
		log.Println(err)
	}

	for _, delay := range s.delays {
		Append(fmt.Sprintf("%s %s %s\n", delay.topic, delay.value, strconv.FormatInt(delay.exectime.Unix(), 10)), "delay.z")
	}
}

func (s *ZSub) reloadDelay() {
	f, err := os.Open("delay.z")
	if err != nil {
		return
	}
	defer f.Close()

	r := bufio.NewReader(f)
	for {
		bytes, err := r.ReadBytes('\n')
		if err != nil {
			return
		}
		line := string(bytes)
		if len(line) == 0 {
			continue
		}
		line = strings.Trim(line, " \r\n")
		split := strings.Split(line, " ")
		if len(split) != 3 {
			continue
		}

		exectime, err := strconv.ParseInt(split[2], 10, 64)
		if err != nil {
			log.Println(err)
			continue
		}
		if exectime < time.Now().Unix() {
			continue
		}
		s.delay([]string{"delay", split[0], split[1], strconv.FormatInt((exectime-time.Now().Unix())*1000, 10)}, nil)
	}
}
