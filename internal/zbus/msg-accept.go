package zbus

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"
	"zhub/internal/auth"
)

var AuthManager *auth.PermissionManager

func init() {
	AuthManager = &auth.PermissionManager{}
	// Initialize the permission manager
	err := AuthManager.Init()
	if err != nil {
		log.Fatal(err)
	}
}

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
	}

	// 准入拦截，所有指令完成 auth 认证后才可进入
	if c.user == 0 && Conf.Service.Auth && rcmd[0] != "auth" {
		c.send("-Auth: NOAUTH Authentication required:" + rcmd[0])
		return
	}
	// 指令预处理
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

	// auth check
	switch cmd {
	case "publish", "broadcast", "delay", "rpc":
		if !AuthManager.AuthCheck(c.user, rcmd[1], "w") {
			c.send("-Error: Insufficient permissions to send " + cmd + " [" + rcmd[1] + "] message.")
			return
		}
	case "subscribe": // 在订阅逻辑处检查
	default: // 其他指令将放行,包括：unsubscribe、lock、unlock、timer
	}

	switch cmd {
	case "auth":
		userid, err := AuthManager.GetUserIdByToken(rcmd[1])
		if err != nil {
			c.send("-Error: " + err.Error())
			return
		}

		c.user = userid
		c.send("+Auth: ok!")

		// hide the auth token content
		str := func() string {
			str := rcmd[1]
			length := len(str)
			if length > 4 {
				return str[:2] + strings.Repeat("*", length-4) + str[length-2:]
			} else if length > 2 {
				return str[:1] + strings.Repeat("*", length-2) + str[length-1:]
			} else {
				return strings.Repeat("*", length)
			}
		}()

		log.Printf("[%d] cmd: %s, auth [OK]\n", v.Conn.sn, str)
		return
	case "groupid":
		c.groupid = rcmd[1]
		return
	case "rpc":
		// if rpc and  no sub back error
		if Bus.noSubscribe(rcmd[1]) {
			rpcBody := make(map[string]string)
			json.Unmarshal([]byte(rcmd[2]), &rpcBody)
			log.Println("rpc no subscribe: ", rcmd[1])

			ruk := rpcBody["ruk"]
			Bus.Publish(strings.Split(ruk, "::")[0], "{'retcode': 404, 'retinfo': '服务离线！', 'ruk': '"+ruk+"'}")
			return
		}

		if len(rcmd) != 3 {
			c.send("-Error: publish para number![" + strings.Join(rcmd, " ") + "]")
		} else {
			/*if len(topicChan) < cap(topicChan) {
				topicChan <- rcmd
			}*/
			Bus.Publish(rcmd[1], rcmd[2])
		}
		return
	case "publish":
		if len(rcmd) != 3 {
			c.send("-Error: publish para number![" + strings.Join(rcmd, " ") + "]")
		} else {
			/*if len(topicChan) < cap(topicChan) {
				topicChan <- rcmd
			}*/
			Bus.Publish(rcmd[1], rcmd[2])
		}
		return
	case "broadcast":
		Bus.broadcast(rcmd[1], rcmd[2])
	case "delay":
		Bus.Delay(rcmd)
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
				// auth check
				if !AuthManager.AuthCheck(c.user, rcmd[1], "r") {
					c.send("-Error: Insufficient permissions to " + cmd + " [" + rcmd[1] + "] message.")
					continue
				}
				c.subscribe(topic)
			}
		case "unsubscribe":
			for _, topic := range rcmd[1:] {
				c.unsubscribe(topic)
			}
		case "timer":
			for _, name := range rcmd[1:] {
				Bus.timer([]string{"timer", name}, c) // append to timers
				c.timers = append(c.timers, name)     // append to conns
			}
		case "cmd":
			if len(rcmd) == 1 {
				return
			}
			switch rcmd[1] {
			case "reload-timer":
				Bus.ReloadTimer()
			case "shutdown":
				if AuthManager.IsAdmin(c.user) {
					return
				}
				Bus.shutdown()
			}
		case "lock":
			// lock key uuid 5
			if len(rcmd) != 4 {
				c.send("-Error: lock para number![" + strings.Join(rcmd, " ") + "]")
				return
			}
			d, _ := strconv.Atoi(rcmd[3])
			Bus._lock(&Lock{key: rcmd[1], uuid: rcmd[2], duration: d})
		case "unlock":
			// unlock key uuid
			if len(rcmd) != 3 {
				c.send("-Error: unlock para number![" + strings.Join(rcmd, " ") + "]")
				return
			}
			Bus._unlock(Lock{key: rcmd[1], uuid: rcmd[2]})
		default:
			c.send("-Error: default not supported:[" + strings.Join(rcmd, " ") + "]")
			return
		}
	}
}
