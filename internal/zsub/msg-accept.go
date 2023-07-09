package zsub

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"
)

var funChan = make(chan func(), 1000)

func handleMessage(v Message) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("handleMessage Recovered:", r)
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

	if Conf.Log.Level == "debug" && rcmd[0] != "auth" {
		log.Printf("[%d] cmd: %s\n", v.Conn.sn, strings.Join(rcmd, " "))
	} else if rcmd[0] == "auth" {
		if len(rcmd) != 2 || strings.IndexAny(rcmd[1], "@") == -1 {
			c.send("-Error: invalid password!")
			return
		}

		inx := strings.IndexAny(rcmd[1], "@") //user@pwd

		authKey := rcmd[1][:inx]              //user
		authValue := Conf.Auth[rcmd[1][:inx]] //pwd
		if strings.EqualFold(authValue, rcmd[1][inx+1:]) {
			c.auth = rcmd[1][:inx]
			c.send("+Auth: ok!")
			log.Printf("[%d] cmd: %s\n", v.Conn.sn, "auth "+authKey+"@******* "+"[OK]")
		} else {
			c.send("-Auth: invalid password!")
			log.Printf("[%d] cmd: %s\n", v.Conn.sn, "auth "+authKey+"@******* "+"[Error]")
		}
		return
	}

	if strings.TrimSpace(c.auth) == "" && rcmd[0] != "auth" && Conf.Service.Auth {
		c.send("-Auth: NOAUTH Authentication required:" + rcmd[0])
		return
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
	case "rpc":
		// if rpc and  no sub back error
		if Hub.noSubscribe(rcmd[1]) {
			rpcBody := make(map[string]string)
			json.Unmarshal([]byte(rcmd[2]), &rpcBody)
			log.Println("rpc no subscribe: ", rcmd[1])

			ruk := rpcBody["ruk"]
			Hub.Publish(strings.Split(ruk, "::")[0], "{'retcode': 404, 'retinfo': '服务离线！', 'ruk': '"+ruk+"'}")
			return
		}

		if len(rcmd) != 3 {
			c.send("-Error: publish para number![" + strings.Join(rcmd, " ") + "]")
		} else {
			/*if len(topicChan) < cap(topicChan) {
				topicChan <- rcmd
			}*/
			Hub.Publish(rcmd[1], rcmd[2])
		}
		return
	case "publish":
		if len(rcmd) != 3 {
			c.send("-Error: publish para number![" + strings.Join(rcmd, " ") + "]")
		} else {
			/*if len(topicChan) < cap(topicChan) {
				topicChan <- rcmd
			}*/
			Hub.Publish(rcmd[1], rcmd[2])
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
			Hub.broadcast(rcmd[1], rcmd[2])
		case "delay":
			Hub.Delay(rcmd)
		case "timer":
			for _, name := range rcmd[1:] {
				Hub.timer([]string{"timer", name}, c) // append to timers
				c.timers = append(c.timers, name)     // append to conns
			}
		case "cmd":
			if len(rcmd) == 1 {
				return
			}
			switch rcmd[1] {
			case "reload-timer":
				Hub.ReloadTimer()
			case "shutdown":
				if !strings.EqualFold(c.groupid, "group-admin") {
					return
				}
				Hub.shutdown()
			}
		case "lock":
			// lock key uuid 5
			if len(rcmd) != 4 {
				c.send("-Error: lock para number![" + strings.Join(rcmd, " ") + "]")
				return
			}
			d, _ := strconv.Atoi(rcmd[3])
			Hub._lock(&Lock{key: rcmd[1], uuid: rcmd[2], duration: d})
		case "unlock":
			// unlock key uuid
			if len(rcmd) != 3 {
				c.send("-Error: unlock para number![" + strings.Join(rcmd, " ") + "]")
				return
			}
			Hub._unlock(Lock{key: rcmd[1], uuid: rcmd[2]})
		/*case "auth":
		if len(rcmd) != 2 || strings.IndexAny(rcmd[1], "@") == -1 {
			c.send("-Error: invalid password!")
			return
		}

		inx := strings.IndexAny(rcmd[1], "@") //user@pwd

		authKey := Conf.Auth[rcmd[1][:inx]]
		if strings.EqualFold(authKey, rcmd[1][inx+1:]) {
			c.auth = rcmd[1][:inx]
			c.send("+Auth: ok!")
		} else {
			c.send("-Auth: invalid password!")
		}
		return*/
		default:
			c.send("-Error: default not supported:[" + strings.Join(rcmd, " ") + "]")
			return
		}
	}
}