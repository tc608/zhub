package zsub

import (
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
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

	log.Println("rcmd: " + strings.Join(rcmd, " "))

	if len(rcmd) == 1 {
		switch strings.ToLower(rcmd[0]) {
		default:
			// subscribe|unsubscribe|daly
			if strings.Index(rcmd[0], "subscribe") == 0 || strings.Index(rcmd[0], "unsubscribe") == 0 || strings.Index(rcmd[0], "daly") == 0 {
				rcmd = strings.Split(rcmd[0], " ")
			} else {
				send(c.conn, "-Error: not supported! (tips: send help)")
				return
			}
		}
	}

	cmd := rcmd[0]
	switch cmd {
	case "subscribe":
		//subscribe x y z
		for _, topic := range rcmd[1:] {
			zsub.subscribe(c, topic) // todo: 批量一次订阅
		}
	case "unsubscribe":
		for _, topic := range rcmd[1:] {
			zsub.unsubscribe(c, topic)
		}
	case "publish":
		if len(rcmd) != 3 {
			send(c.conn, "-Error: publish para number!")
		} else {
			zsub.publish(rcmd[1], rcmd[2])
		}
	case "daly":
		daly(rcmd, c)
	case "timer":
		// todo Timer(rcmd, conn)
	default:
		send(c.conn, "-Error: default not supported:["+strings.Join(rcmd, " ")+"]")
		return
	}
}

// daly topic valye 100
func daly(rcmd []string, c *ZConn) {
	if len(rcmd) != 4 {
		send(c.conn, "-Error: subscribe para number!")
		return
	}

	t, err := strconv.ParseInt(rcmd[3], 10, 64)
	if err != nil {
		send(c.conn, "-Error: "+strings.Join(rcmd, " "))
		return
	}

	timer := time.NewTimer(time.Duration(t) * time.Millisecond)
	select {
	case <-timer.C:
		zsub.publish(rcmd[1], rcmd[2])
	}
}

var wlock = sync.Mutex{}

// 发送消息
func send(conn *net.Conn, vs ...string) error {
	wlock.Lock()
	defer wlock.Unlock()

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
	_, err := (*conn).Write(bytes)
	return err
}
