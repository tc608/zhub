package zsub

import (
	"log"
	"strconv"
	"strings"
	"time"
	"zhub/conf"
)

func msgAccept(v Message) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("ExecCmd Recovered:", r)
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

			arr := []string{"subscribe", "unsubscribe", "daly", "groupid"}
			if startWithAny(rcmd[0], arr...) {
				rcmd = strings.Split(rcmd[0], " ")
			} else {
				c.send("-Error: not supported:" + rcmd[0])
				return
			}
		}
	}

	cmd := rcmd[0]
	switch cmd {
	case "groupid":
		c.groupid = rcmd[1]
	case "subscribe":
		// subscribe x y z
		for _, topic := range rcmd[1:] {
			zsub.subscribe(c, topic) // todo: 批量一次订阅
		}
	case "unsubscribe":
		for _, topic := range rcmd[1:] {
			zsub.unsubscribe(c, topic)
		}
	case "publish":
		if len(rcmd) != 3 {
			c.send("-Error: publish para number!")
		} else {
			zsub.publish(rcmd[1], rcmd[2])
		}
	case "daly":
		daly(rcmd, c)
	case "timer":
		zsub.timer(rcmd, c)
	case "cmd":
		if len(rcmd) == 1 {
			return
		}
		switch rcmd[1] {
		case "reload-timer-config":
			zsub.reloadTimerConfig()
		}
	default:
		c.send("-Error: default not supported:[" + strings.Join(rcmd, " ") + "]")
		return
	}
}

// daly topic value 100 -> publish topic value
func daly(rcmd []string, c *ZConn) {
	if len(rcmd) != 4 {
		c.send("-Error: subscribe para number!")
		return
	}

	t, err := strconv.ParseInt(rcmd[3], 10, 64)
	if err != nil {
		c.send("-Error: " + strings.Join(rcmd, " "))
		return
	}

	timer := time.NewTimer(time.Duration(t) * time.Millisecond)
	select {
	case <-timer.C:
		zsub.publish(rcmd[1], rcmd[2])
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
