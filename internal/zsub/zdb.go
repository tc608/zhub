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
	datadir string
)

// Message 数据封装
type Message struct {
	Conn *ZConn
	Rcmd []string
}

// Append file append
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

func (s *ZSub) SaveData() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("SaveData Recovered:", r)
		}
	}()

	s.Lock()
	defer s.Unlock()
	// ========================== delay save ===========================
	func() {
		if !s.delayup {
			return
		}
		defer func() {
			s.delayup = false
		}()

		err := os.Remove(datadir + "/delay.z")
		if err != nil {
			log.Println(err)
		}
		file, err := os.OpenFile(datadir+"/delay.z", os.O_CREATE|os.O_WRONLY, os.ModeAppend)
		if err != nil {
			fmt.Println(err)
		}
		defer file.Close()

		writer := bufio.NewWriter(file)
		_delays := s.delays

		for _, delay := range _delays {
			delayStr := fmt.Sprintf("%s %s %d\n", delay.Topic, delay.Value, delay.Exectime.Unix())
			writer.WriteString(delayStr)
		}
		writer.Flush()
	}()

	// ========================== lock save ===========================
	func() {
		err := os.Remove(datadir + "/lock.z")
		if err != nil {
			log.Println(err)
		}
		str := ""
		for _, locks := range s.locks {
			for _, lock := range locks {
				str += fmt.Sprintf("%s %s %d %d\n", lock.key, lock.uuid, lock.duration, lock.start)
				break // 只记录获得锁的记录
			}
		}
		Append(str, datadir+"/lock.z")
	}()
}

func (s *ZSub) LoadData() {
	s.loadDelay()
	// s.loadLock()
}

func (s *ZSub) loadDelay() {
	f, err := os.Open(datadir + "/delay.z")
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
	f, err := os.Open(datadir + "/lock.z")
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
