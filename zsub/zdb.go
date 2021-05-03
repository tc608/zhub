package zsub

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
	"zhub/conf"
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
func (s *ZSub) dataStorage() {
	s.Lock()
	defer s.Unlock()
	// delay save
	err := os.Remove(conf.DataDir + "/delay.z")
	if err != nil {
		log.Println(err)
	}

	var str string
	for _, delay := range s.delays {
		str += fmt.Sprintf("%s %s %s\n", delay.topic, delay.value, strconv.FormatInt(delay.exectime.Unix(), 10))
	}
	Append(str, conf.DataDir+"/delay.z")

	// lock save
	err = os.Remove(conf.DataDir + "/lock.z")
	if err != nil {
		log.Println(err)
	}
	str = ""
	for _, locks := range s.locks {
		for _, lock := range locks {
			str += fmt.Sprintf("%s %s %d %d\n", lock.key, lock.uuid, lock.duration, lock.start)
			break // 只记录获得锁的记录
		}
	}
	Append(str, conf.DataDir+"/lock.z")
}

func (s *ZSub) loadDelay() {
	f, err := os.Open(conf.DataDir + "/delay.z")
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

func (s *ZSub) loadLock() {
	f, err := os.Open(conf.DataDir + "/lock.z")
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
		if len(split) != 4 {
			continue
		}
		duration, err := strconv.Atoi(split[2])
		start, err := strconv.ParseInt(split[3], 10, 64)

		if start > 0 && time.Now().Unix()-start > 1 {
			duration = int(time.Now().Unix() - start)
		} else {
			duration = 1
		}

		s._lock(&Lock{
			key:      split[0],
			uuid:     split[1],
			duration: duration,
			// start:    start,
		})
	}
}
