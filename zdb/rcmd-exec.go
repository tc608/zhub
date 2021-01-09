package _zdb

import (
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

func ExecCmd(rcmd []string, conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("ExecCmd Recovered:", r)
		}
	}()
	if len(rcmd) == 0 {
		return
	}

	log.Println("rcmd: " + strings.Join(rcmd, " "))

	if len(rcmd) == 1 {
		switch strings.ToLower(rcmd[0]) {
		case "help":
			conn.Write([]byte("help-start\r\n"))
			conn.Write(retHelp)
			conn.Write([]byte("help-end\r\n"))
			return
		default:
			// subscribe|unsubscribe|daly
			if strings.Index(rcmd[0], "subscribe") == 0 || strings.Index(rcmd[0], "unsubscribe") == 0 || strings.Index(rcmd[0], "daly") == 0 {
				rcmd = strings.Split(rcmd[0], " ")
			} else {
				conn.Write([]byte("-Error: not supported! (tips: send help)\r\n"))
				return
			}
		}
	}

	cmd := rcmd[0]
	switch cmd {
	case "decr":
		decr(rcmd, conn)
	case "incr":
		incr(rcmd, conn)
	case "get":
		get(rcmd, conn)
	case "set":
		set(rcmd, conn)
	case "subscribe":
		subscribe(rcmd, conn)
	case "unsubscribe":
		unsubscribe(rcmd, conn)
	case "publish":
		publish(rcmd, conn)
	case "daly":
		daly(rcmd, conn)
	case "timer":
		timer(rcmd, conn)
	default:
		conn.Write([]byte("-Error: default not supported:[" + strings.Join(rcmd, " ") + "]\r\n"))
		return
	}
}

// daly topic valye 100
func daly(rcmd []string, conn net.Conn) {
	if len(rcmd) != 4 {
		conn.Write([]byte("-Error: subscribe para number!\r\n"))
		return
	}

	t, err := strconv.ParseInt(rcmd[3], 10, 64)
	if err != nil {
		conn.Write([]byte("-Error: " + strings.Join(rcmd, " ") + "\r\n"))
		return
	}

	timer := time.NewTimer(time.Duration(t) * time.Millisecond)
	select {
	case <-timer.C:
		// daly => publish
		publish(rcmd[0:3], conn)
	}
}

func decr(rcmd []string, conn net.Conn) {
	k := rcmd[1]
	v := zkv[k]
	if strings.EqualFold(v, "") {
		v = "0"
	}
	_v, err := strconv.Atoi(v)
	if err != nil {
		conn.Write([]byte("-Error: " + err.Error() + "\r\n"))
	}

	v = strconv.Itoa(_v - 1)
	zkv[k] = v
	conn.Write([]byte(v + "\r\n"))
}

func incr(rcmd []string, conn net.Conn) {
	k := rcmd[1]
	v := zkv[k]
	if strings.EqualFold(v, "") {
		v = "0"
	}
	_v, err := strconv.Atoi(v)
	if err != nil {
		conn.Write([]byte("- Error: " + err.Error() + "\r\n"))
	}

	v = strconv.Itoa(_v + 1)
	zkv[k] = v
	conn.Write([]byte(v + "\r\n"))
}

func get(rcmd []string, conn net.Conn) {
	k := rcmd[1]
	v := zkv[k]
	conn.Write([]byte(v + "\r\n"))
}

func set(rcmd []string, conn net.Conn) {
	if len(rcmd) != 3 {
		conn.Write([]byte("-Error: set para number!\r\n"))
		return
	}
	zkv[rcmd[1]] = rcmd[2]
	conn.Write([]byte("+OK\r\n"))
}

func subscribe(rcmd []string, conn net.Conn) {
	if len(rcmd) < 2 {
		conn.Write([]byte("-Error: subscribe para number!\r\n"))
		return
	}

	for _, topic := range rcmd[1:] {
		conns := zsub[topic]
		if conns == nil {
			conns = make([]*ConnContext, 0)
		}

		zsub[topic] = append(conns, &ConnContext{conn: &conn})
	}
}
func unsubscribe(rcmd []string, conn net.Conn) {
	if len(rcmd) < 2 {
		conn.Write([]byte("-Error: unsubscribe para number!"))
		return
	}

	for _, topic := range rcmd[1:] {
		conns := zsub[topic]
		if conns == nil || len(conns) == 0 {
			return
		}
		_conns := make([]*ConnContext, 0)
		for _, c := range conns {
			if *c.conn == *&conn {
				continue
			}
			_conns = append(_conns, c)
		}
		zsub[topic] = _conns
	}
}
func publish(rcmd []string, conn net.Conn) {
	if len(rcmd) < 3 {
		conn.Write([]byte("-Error: publish para number!\r\n"))
		return
	}

	topic := rcmd[1]
	v := rcmd[2]

	subs := zsub[topic]
	if subs == nil || len(subs) == 0 {
		return
	}

	msgs := []string{"message", topic, v}
	for _, c := range subs {
		Send(*c.conn, msgs...)
		/*_conn.Write([]byte("*3\r\n"))
		for _, msg := range msgs {
			_conn.Write([]byte("$" + strconv.Itoa(len(msg)) + "\r\n"))
			_conn.Write([]byte(msg + "\r\n"))
		}*/
	}
}

var wlock = sync.Mutex{}

func Send(conn net.Conn, vs ...string) (err error) {
	//chSend <- vs
	wlock.Lock()
	defer wlock.Unlock()

	if len(vs) == 1 {
		_, err = conn.Write([]byte(vs[0] + "\r\n"))
	} else if len(vs) > 1 {
		data := "*" + strconv.Itoa(len(vs)) + "\r\n"
		for _, v := range vs {
			data += "$" + strconv.Itoa(len(v)) + "\r\n"
			data += v + "\r\n"
		}
		_, err = conn.Write([]byte(data))
	}

	return err
}
