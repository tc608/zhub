package zsub

import (
	"log"
	"strconv"
	"strings"
	"time"
	"zhub/conf"
)

var funChan = make(chan func(), 1000)

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

	// ping reply
	if strings.EqualFold("+pong", v.Rcmd[0]) {
		v.Conn.pong = time.Now().Unix()
		return
	}

	if conf.LogDebug {
		log.Println("[", v.Conn.sn, "] rcmd: "+strings.Join(rcmd, " "))
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

			arr := []string{"subscribe", "timer", "unsubscribe", "delay", "groupid"}
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
		return
	case "publish":
		if len(rcmd) != 3 {
			c.send("-Error: publish para number![" + strings.Join(rcmd, " ") + "]")
		} else {
			zsub.publish(rcmd[1], rcmd[2])
		}
		return
	default:
	}

	// 内部执行指令 加入执行队列
	funChan <- func() {
		defer func() {
			if r := recover(); r != nil {
				log.Println("funChan Recovered:", r)
			}
		}()
		switch cmd {
		case "subscribe":
			// subscribe x y z
			for _, topic := range rcmd[1:] {
				c.subscribe(topic) // todo: 批量一次订阅
			}
		case "unsubscribe":
			for _, topic := range rcmd[1:] {
				c.unsubscribe(topic)
			}
		case "broadcast":
			zsub.broadcast(rcmd[1], rcmd[2])
		case "delay":
			zsub.delay(rcmd, c)
		case "timer":
			for _, name := range rcmd[1:] {
				zsub.timer([]string{"timer", name}, c)
			}
		case "cmd":
			if len(rcmd) == 1 {
				return
			}
			switch rcmd[1] {
			case "reload-timer":
				zsub.ReloadTimer()
			case "shutdown":
				if !strings.EqualFold(c.groupid, "group-admin") {
					return
				}
				zsub.shutdown()
			}
		case "lock":
			// lock key uuid 5
			if len(rcmd) != 4 {
				c.send("-Error: lock para number![" + strings.Join(rcmd, " ") + "]")
				return
			}
			d, _ := strconv.Atoi(rcmd[3])
			zsub._lock(&Lock{key: rcmd[1], uuid: rcmd[2], duration: d})
		case "unlock":
			// unlock key uuid
			if len(rcmd) != 3 {
				c.send("-Error: unlock para number![" + strings.Join(rcmd, " ") + "]")
				return
			}
			zsub._unlock(Lock{key: rcmd[1], uuid: rcmd[2]})
		default:
			c.send("-Error: default not supported:[" + strings.Join(rcmd, " ") + "]")
			return
		}
	}
}
