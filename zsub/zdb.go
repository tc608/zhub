package zsub

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

/*
var (
	topicChan = make(chan []string, 1000) //接收到的 所有消息数据, 用于写入数据库持久化
)*/

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

// 数据持久化
func (s *ZSub) dataStorage() {
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

		err := os.Remove(DataDir + "/delay.z")
		if err != nil {
			log.Println(err)
		}
		file, err := os.OpenFile(DataDir+"/delay.z", os.O_CREATE|os.O_WRONLY, os.ModeAppend)
		if err != nil {
			fmt.Println(err)
		}
		defer file.Close()
		writer := bufio.NewWriter(file)
		delays2 := s.delays

		for _, delay := range delays2 {
			writer.WriteString(delay.topic)
			writer.WriteString(" ")
			writer.WriteString(delay.value)
			writer.WriteString(" ")
			writer.WriteString(strconv.FormatInt(delay.exectime.Unix(), 10))
			writer.WriteString("\n")
		}
		writer.Flush()
	}()

	// ========================== lock save ===========================
	func() {
		err := os.Remove(DataDir + "/lock.z")
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
		Append(str, DataDir+"/lock.z")
	}()
}

func (s *ZSub) loadDelay() {
	f, err := os.Open(DataDir + "/delay.z")
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
	f, err := os.Open(DataDir + "/lock.z")
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

// --------------------------------------
var (
	db  *sql.DB
	seq int64 = 50000
)

func init() {
	LoadConf("app.conf")
	/*
		_db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8",
			GetStr("ztimer.db.user", "root"),
			GetStr("ztimer.db.pwd", "123456"),
			GetStr("ztimer.db.addr", "127.0.0.1:3306"),
			GetStr("ztimer.db.database", "zhub"),
		))
		if err != nil {
			log.Println(err)
			return
		}

		db = _db

		// 批量写入数据库，等待超时5秒，如有数据写入数据

		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Println("MsgToDb Recovered:", r)
				}
			}()

			var flagcount = 0
			var _sql = "INSERT INTO zhub.topicmessage (`msgid`,`topic`,`value`,`createtime`) VALUES \n"
			for {
				select {
				case msg := <-topicChan:
					var topic, value = msg[1], msg[2]
					var t = time.Now().UnixNano() / 1e6
					_sql += fmt.Sprintf("('%s','%s','%s',%d),\n",
						strconv.FormatInt(t, 36)+"-"+strconv.FormatInt(atomic.AddInt64(&seq, 1), 36), topic, value, t)
					flagcount++
				case <-time.After(time.Second * 5): // 等待5秒
					if flagcount > 0 {
						flagcount = 100
					}
				}

				if flagcount != 100 {
					continue
				}

				_sql = _sql[:len(_sql)-2]
				_sql += ";"

				_, err = db.Exec(_sql)
				if err != nil {
					log.Println(err)
				}

				_sql = "INSERT INTO zhub.topicmessage (`msgid`,`topic`,`value`,`createtime`) VALUES \n"
				flagcount = 0
			}
		}()
	*/
}
