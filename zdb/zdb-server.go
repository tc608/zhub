package zdb

import (
	"bufio"
	"fmt"
	"github.com/robfig/cron"
	"log"
	"net"
	"strconv"
	"time"
)

// 消息命令处理 chan
var (
	chmsg   = make(chan Message, 10000)
	zkv     = make(map[string]string)
	zsub    = make(map[string][]*ConnContext) // topic -- connx[]
	retOk   = []byte("+OK")
	zTimer  = make(map[string]*ZTimer)
	retHelp = []byte(
		"\n--- zdb help ---\n" +
			"______  _____   _____  \n|___  / |  _  \\ |  _  \\ \n   / /  | | | | | |_| | \n  / /   | | | | |  _  { \n / /__  | |_| | | |_| | \n/_____| |_____/ |_____/ \n" +
			"had supported command:\n" +
			"1. set:\n" +
			" eg: set a 1\n" +
			"2. get:\n" +
			" eg: get a\n" +
			"3. subscribe:\n" +
			" eg: subscribe x y z\n" +
			"4. unsubscribe:\n" +
			" eg: unsubscribe x1 y1 z1\n" +
			"5. publish:\n" +
			" eg: publish x 123\n" +
			"6. incr:\n" +
			" eg: incr a\n" +
			"7. decr:\n" +
			" eg: decr a\n" +
			"--- zdb help ---\n")
)

// 数据封装
type Message struct {
	Conn *net.Conn
	Rcmd []string
}
type ConnContext struct {
	conn       *net.Conn
	groupId    string
	createTime time.Time
}

type ZTimer struct {
	conns []*net.Conn
	expr  string
	topic string
	cron  *cron.Cron
}

// ======================================================================

// zdb 服务启动
func ServerStart(host string, port int) {
	listen, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Printf("zdb started listen on: %s:%d \n", host, port)

	// 启动消息监听处理
	go func() {
		for {
			v, ok := <-chmsg
			if !ok {
				break
			}

			execCmd(v.Rcmd, *&*v.Conn)
		}
	}()

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		fmt.Println("conn start: ", conn.RemoteAddr())

		go connHandler(conn)
	}
}

// 连接处理
func connHandler(conn net.Conn) {
	defer func() {
		for topic, connx := range zsub {
			_conns := make([]*ConnContext, 0)
			for t := range connx {
				if *connx[t].conn == *&conn {
					continue
				}
				_conns = append(_conns, connx[t])
			}
			zsub[topic] = _conns
		}
		conn.Close()
		if r := recover(); r != nil {
			log.Println("connHandler Recovered:", r)
		}

		fmt.Println("conn end: ", conn.RemoteAddr())
	}()

	reader := bufio.NewReader(conn)

	for {
		rcmd := make([]string, 0)
		line, _, err := reader.ReadLine()
		// fmt.Println("line:", string(line)) todo 可使用第一行用于协议头
		if err != nil {
			log.Println(err)
			return
		}
		if len(line) == 0 {
			continue
		}
		switch string(line[:1]) {
		case "*":
			n, _ := strconv.Atoi(string(line[1:]))
			for i := 0; i < n; i++ {
				reader.ReadLine()
				v, _, _ := reader.ReadLine()
				rcmd = append(rcmd, string(v))
			}
		default:
			rcmd = append(rcmd, string(line))
		}

		if len(rcmd) == 0 {
			continue
		}

		chmsg <- Message{Conn: &conn, Rcmd: rcmd}
	}
}
