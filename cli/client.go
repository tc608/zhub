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

type Client struct {
	wlock sync.Mutex // write lock
	rlock sync.Mutex // read lock

	addr       string    // host:port
	conn       net.Conn  // socket conn
	createTime time.Time // client create time
	groupid    string    // client group id

	subFun   map[string]func(v string) // subscribe topic and callback function
	timerFun map[string]func()         // subscribe timer amd callback function

	chSend       chan []string // chan of send message
	chReceive    chan []string // chan of receive message
	timerReceive chan []string // chan of timer
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
		groupid:    groupid,
		createTime: time.Now(),

		subFun:       make(map[string]func(v string)),
		timerFun:     make(map[string]func()),
		chSend:       make(chan []string, 100),
		chReceive:    make(chan []string, 100),
		timerReceive: make(chan []string, 100),
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
			conn.Write([]byte("groupid " + c.groupid + "\r\n"))
			go c.receive()

			// 重新订阅
			for topic, _ := range c.subFun {
				c.subscribes(topic)
			}
			for topic, _ := range c.timerFun {
				c.timer(topic)
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
			case vs := <-c.chReceive:
				fun := c.subFun[vs[1]]
				if fun == nil {
					log.Println("topic received, nothing to do", vs[1], vs[2])
					continue
				}
				fun(vs[2])
			case vs := <-c.timerReceive:
				fun := c.timerFun[vs[1]]
				if fun == nil {
					log.Println("timer received, nothing to do", vs[1])
					continue
				}
				fun()
			}

		}
	}()

	go c.receive()
}

func (c *Client) Subscribe(topic string, fun func(v string)) {
	c.subFun[topic] = fun
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
send topic message :
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

func (c *Client) Timer(topic string, expr string, fun func()) {
	c.timerFun[topic] = fun
	c.send("timer", topic, expr)
}

// todo: save client timer‘s info
func (c *Client) timer(topic string) {
	c.send("timer", topic)
}

/*
// subscribe topic
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

/*
send socket message :
if len(vs) equal 1 will send message `vs[0] + "\r\n"`
else if len(vs) gt 1 will send message `* + len(vs)+ "\r\n"  +"$"+ len(vs[n])+ "\r\n" + vs[n] + "\r\n" ...`
*/
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
				c.chReceive <- vs
				continue
			}
			if len(vs) == 2 && strings.EqualFold(vs[0], "timer") {
				c.timerReceive <- vs
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

func (c *Client) get(key string) string {

	return ""
}

// -------------------------------------- hm --------------------------------------

// ==============================================================================

var reconnect = 0

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
