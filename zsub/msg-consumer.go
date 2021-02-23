package zsub

import (
	"log"
	"strconv"
	"strings"
	"time"
	"zhub/conf"
)

var funChan = make(chan func(), 1000)

type ZDelay struct {
	topic    string
	value    string
	exectime time.Time
	timer    *time.Timer
}

func msgAccept(v Message) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("msgAccept Recovered:", r)
		}
	}()
	c := v.Conn
	rcmd := v.Rcmd

	if len(rcmd) == 0 {
		return
	}

	if conf.LogDebug {
		log.Println("rcmd: " + strings.Join(rcmd, " "))
	}

	if len(rcmd) == 1 {
		switch strings.ToLower(rcmd[0]) {
		default:
			// str start with strs anyone
			var startWithAny = func(str string, strs ...string) bool {
				for _, str := range strs {
					if strings.Index(rcmd[0], str) == 0 {
						return true
					}
				}
				return false
			}

			arr := []string{"subscribe", "unsubscribe", "delay", "groupid"}
			if startWithAny(rcmd[0], arr...) {
				rcmd = strings.Split(rcmd[0], " ")
			} else {
				c.send("-Error: not supported:" + rcmd[0])
				return
			}
		}
	}

	cmd := rcmd[0]
	if strings.EqualFold(cmd, "groupid") {
		c.groupid = rcmd[1]
		return
	} else if strings.EqualFold(cmd, "publish") {
		if len(rcmd) != 3 {
			c.send("-Error: publish para number!")
		} else {
			zsub.publish(rcmd[1], rcmd[2])
		}
		return
	}

	// 内部执行指令 加入执行队列
	funChan <- func() {
		switch cmd {
		case "subscribe":
			// subscribe x y z
			for _, topic := range rcmd[1:] {
				zsub.subscribe(c, topic) // todo: 批量一次订阅
			}
		case "unsubscribe":
			for _, topic := range rcmd[1:] {
				zsub.unsubscribe(c, topic)
			}
		case "broadcast":
			zsub.broadcast(rcmd[1], rcmd[2])
		case "delay":
			zsub.delay(rcmd, c)
		case "timer":
			zsub.timer(rcmd, c)
		case "cmd":
			if len(rcmd) == 1 {
				return
			}
			switch rcmd[1] {
			case "reload-timer-config":
				zsub.reloadTimerConfig()
			case "shutdown":
				if !strings.EqualFold(c.groupid, "group-admin") {
					return
				}
				zsub.shutdown()
			}
		default:
			c.send("-Error: default not supported:[" + strings.Join(rcmd, " ") + "]")
			return
		}
	}
}

// delay topic value 100 -> publish topic value
func (s *ZSub) delay(rcmd []string, c *ZConn) {
	s.Lock()
	defer func() {
		s.Unlock()
		s.saveDelay()
	}()
	if len(rcmd) != 4 {
		c.send("-Error: subscribe para number!")
		return
	}

	t, err := strconv.ParseInt(rcmd[3], 10, 64)
	if err != nil {
		c.send("-Error: " + strings.Join(rcmd, " "))
		return
	}

	delay := s.delays[rcmd[1]+"-"+rcmd[2]]
	if delay != nil {
		if t == -1 {
			delay.timer.Stop()
			delete(s.delays, rcmd[1]+"-"+rcmd[2])
			return
		}
		delay.timer.Reset(time.Duration(t) * time.Millisecond)
	} else {
		delay := &ZDelay{
			topic:    rcmd[1],
			value:    rcmd[2],
			exectime: time.Now().Add(time.Duration(t) * time.Millisecond),
			timer:    time.NewTimer(time.Duration(t) * time.Millisecond),
		}
		s.delays[rcmd[1]+"-"+rcmd[2]] = delay
		go func() {
			select {
			case <-delay.timer.C:
				zsub.publish(rcmd[1], rcmd[2])
				delete(s.delays, rcmd[1]+"-"+rcmd[2])
			}
		}()
	}
}

// send message
func (c *ZConn) send(vs ...string) error {
	c.Lock()
	defer c.Unlock()

	var bytes []byte

	if len(vs) == 1 {
		bytes = []byte(vs[0] + "\r\n")
	} else if len(vs) > 1 {
		data := "*" + strconv.Itoa(len(vs)) + "\r\n"
		for _, v := range vs {
			data += "$" + strconv.Itoa(len(v)) + "\r\n"
			data += v + "\r\n"
		}
		bytes = []byte(data)
	}
	_, err := (*c.conn).Write(bytes)
	return err
}
