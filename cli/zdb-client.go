package cli

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	reconnect    = 0
	subEvent     = make(map[string]func(v string))
	chReceive    = make(chan []string, 1000)
	chSend       = make(chan []string, 1000)
	timerReceive = make(chan []string, 1000)
)

type Client struct {
	wlock      sync.Mutex // 写锁
	rlock      sync.Mutex // 读锁
	addr       string     // host:port
	conn       net.Conn   // socket 连接对象
	createTime time.Time  // 创建时间
}

func Create(addr string, groupid string) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return &Client{}, err
	}

	client := Client{
		wlock:      sync.Mutex{},
		rlock:      sync.Mutex{},
		addr:       addr,
		conn:       conn,
		createTime: time.Now(),
	}

	conn.Write([]byte("groupid " + groupid + "\r\n"))
	client.init()
	return &client, err
}

func (c *Client) reconn() (err error) {
	for n := 1; n < 10; n++ {
		conn, err := net.Dial("tcp", c.addr)
		if err != nil {
			log.Println("reconn", err)
			time.Sleep(time.Second * 3)
			continue
		} else if err == nil {
			c.conn = conn
			go c.receive()
			for topic, _ := range subEvent {
				c.subscribes(topic)
			}
			break
		}
	}
	return
}

func (c *Client) init() {
	// 消费 topic 消息
	go func() {
		for {
			select {
			case vs := <-chReceive:
				fun := subEvent[vs[1]]
				if fun == nil {
					log.Println("topic received, nothing to do", vs[1], vs[2])
					continue
				}
				fun(vs[2])
			case vs := <-timerReceive:
				log.Println("收到 timer 消息 ", vs[1])
			}

		}

		/*for {
			vs, ok := <-chReceive
			if !ok {
				break
			}

			fun := subEvent[vs[1]]
			if fun == nil {
				log.Println("topic received, nothing to do", vs[1], vs[2])
				continue
			}
			fun(vs[2])
		}*/
	}()

	go c.receive()
}

func (c *Client) Subscribe(topic string, fun func(v string)) {
	subEvent[topic] = fun
	c.subscribes(topic)
}

/*
---
ping
---
*/
func (c *Client) ping() {
	c.send("ping")
}

// -------------------------------------- pub-sub --------------------------------------
/*
发送 主题消息
---
*3
$7
message
$8
my-topic
$24
{username:xx,mobile:xxx}
---
*/
func (c *Client) Publish(topic string, message string) error {
	c.send("publish", topic, message)
	return nil
}

func (c *Client) Daly(topic string, message string, daly int) error {
	c.send("daly", topic, message, strconv.Itoa(daly))
	return nil
}

func (c *Client) Timer(topic string, expr string) {
	c.send("timer", topic, expr)
}

/*
// 订阅主题消息
---
subscribe x y z
---
*/
func (c *Client) subscribes(topics ...string) error {
	if len(topics) == 0 {
		return nil
	}

	messages := "subscribe"
	for _, topic := range topics {
		messages += " " + topic
	}
	c.send(messages)
	return nil
}

// 发送 socket 消息
func (c *Client) send(vs ...string) (err error) {
	//chSend <- vs
	c.wlock.Lock()
	defer c.wlock.Unlock()
a:
	if len(vs) == 1 {
		_, err = c.conn.Write([]byte(vs[0] + "\r\n"))
	} else if len(vs) > 1 {
		data := "*" + strconv.Itoa(len(vs)) + "\r\n"
		for _, v := range vs {
			data += "$" + strconv.Itoa(len(v)) + "\r\n"
			data += v + "\r\n"
		}
		_, err = c.conn.Write([]byte(data))
	}
	if err != nil {
		log.Println(err)
		time.Sleep(time.Second * 3)
		goto a
	}

	return err
}

func (c *Client) receive() {
	c.rlock.Lock()
	defer c.rlock.Unlock()

	r := bufio.NewReader(c.conn)
	for {
		v, _, err := r.ReadLine()
		if err != nil {
			log.Println("receive error and reconn: ", err)
			if err = c.reconn(); err == nil {
				r = bufio.NewReader(c.conn)
			} else {

			}
			time.Sleep(time.Second * 3)
			continue
		} else if len(v) == 0 {
			log.Println("receive empty")
			continue
		}

		switch string(v[0:1]) {
		case "*": // 订阅消息
			// 数据行数
			vlen, err := strconv.Atoi(string(v[1:]))
			if err != nil {
				log.Println("receive parse len error: ", err, string(v))
				continue
			}

			// 读取完整数据
			vs := make([]string, 0)
			for i := 0; i < vlen; i++ {
				r.ReadLine() // $x
				v, _, err = r.ReadLine()
				if err != nil {
					log.Println("receive parse v error: ", err)
				}
				vs = append(vs, string(v))
			}

			if len(vs) == 3 && strings.EqualFold(vs[0], "message") {
				chReceive <- vs
				continue
			}
			if len(vs) == 2 && strings.EqualFold(vs[0], "timer") {
				timerReceive <- vs
				continue
			}

			continue
		case "+": // +pong, +xxx
		case "-":
			fmt.Println("error:", string(v))
		case ":":

		}
	}

}

// -------------------------------------- k-v --------------------------------------
/*
*3
$3
set
$n
x
$m
xx
*/
func (c *Client) set(key string, value interface{}) error {

	return nil
}

/*

 */
func (c *Client) get(key string) string {

	return ""
}

// -------------------------------------- hm --------------------------------------

// ==============================================================================

// client 命令行程序
func ClientRun(host string, port int) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))

	for {
		if err != nil {
			log.Println(err)
			time.Sleep(time.Second * 3)
			conn, err = net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
			continue
		}

		fmt.Println(fmt.Sprintf("had connected server: %s:%d", host, port))
		break
	}

	defer func() {
		if reconnect == 1 {
			conn.Close()
			ClientRun(host, port)
		}
	}()

	go clientRead(conn)

	for {
		inReader := bufio.NewReader(os.Stdin)
		line, err := inReader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			return
		} else if reconnect == 1 {
			return
		}

		line = strings.Trim(line, "\r\n")
		line = strings.Trim(line, "\n")
		line = strings.Trim(line, " ")

		if strings.EqualFold(line, "") {
			continue
		} else if strings.EqualFold(line, ":exit") {
			fmt.Println("exit!")
			return
		}

		//fmt.Println("发送数据：" + line)

		line = strings.ReplaceAll(line, "  ", "")
		parr := strings.Split(line, " ")
		conn.Write([]byte("*" + strconv.Itoa(len(parr)) + "\r\n"))
		for i := range parr {
			conn.Write([]byte("$" + strconv.Itoa(len(parr[i])) + "\r\n"))
			conn.Write([]byte(parr[i] + "\r\n"))
		}
	}
}

func clientRead(conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered:", r)
		}
		reconnect = 1
	}()

	reader := bufio.NewReader(conn)
	for {
		rcmd := make([]string, 0)
		line, _, err := reader.ReadLine()
		if err != nil {
			log.Println("connection error: ", err)
			return
		} else if len(line) == 0 {
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
		case "+":
			rcmd = append(rcmd, string(line))
		case "-":
			rcmd = append(rcmd, string(line))
		case ":":
			rcmd = append(rcmd, string(line))
		case "h":
			if strings.EqualFold(string(line), "help-start") {
				for {
					v, _, _ := reader.ReadLine()
					if strings.EqualFold(string(v), "help-end") {
						break
					}
					rcmd = append(rcmd, string(v)+"\r\n")
				}
			}
		default:
			rcmd = append(rcmd, string(line))
		}

		fmt.Println(">", strings.Join(rcmd, " "))
	}
}
