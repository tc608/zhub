package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/go-basic/uuid"
	"io"
	"unicode/utf8"

	//"github.com/go-basic/uuid"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ZHubClient struct {
	wlock sync.Mutex // write lock
	rlock sync.Mutex // read lock

	appname    string    // local appname
	addr       string    // host:port
	conn       net.Conn  // socket conn
	createTime time.Time // client create time
	groupid    string    // client group id

	subFun   map[string]func(v string) // subscribe topic and callback function
	timerFun map[string]func()         // subscribe timer amd callback function

	chSend       chan []string    // chan of send message
	chReceive    chan []string    // chan of receive message
	timerReceive chan []string    // chan of timer
	lockFlag     map[string]*Lock // chan of lock
	auth         string
}

type Lock struct {
	Key      string   // lock Key
	Value    string   // lock Value
	flagChan chan int //
	// starttime uint32 // lock start time
	// duration  int    // lock duration
}

func (c *ZHubClient) Initx(appname, addr, groupid, auth string) error {
	c.appname = appname
	c.addr = addr
	c.groupid = groupid
	c.auth = auth
	return c.Init()
}

// Init 创建一个客户端
func (c *ZHubClient) Init( /*appname, addr, groupid, auth string*/ ) error {
	conn, err := net.Dial("tcp", c.addr)
	if err != nil {
		return err
	}

	c.conn = conn
	c.wlock = sync.Mutex{}
	c.rlock = sync.Mutex{}
	c.createTime = time.Now()

	c.subFun = make(map[string]func(v string))
	c.timerFun = make(map[string]func())
	c.chSend = make(chan []string, 100)
	c.chReceive = make(chan []string, 100)
	c.timerReceive = make(chan []string, 100)
	c.lockFlag = make(map[string]*Lock)

	c.send("auth", c.auth)
	c.send("groupid " + c.groupid)
	c.init()
	return err
}

func (c *ZHubClient) reconn() (err error) {
	for n := 1; n < 10; n++ {
		conn, err := net.Dial("tcp", c.addr)
		if err != nil {
			log.Println("reconn", err)
			time.Sleep(time.Second * 3)
			continue
		} else if err == nil {
			c.conn = conn
			c.send("auth", c.auth)
			c.send("groupid " + c.groupid)
			go c.receive()

			// 重新订阅
			for topic := range c.subFun {
				c.Subscribe(topic, nil)
			}
			for topic := range c.timerFun {
				c.Timer(topic, nil)
			}
			break
		}
	}
	return
}

func (c *ZHubClient) init() {
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
	c.rpcInit()

	go c.receive()
}

// Subscribe subscribe topic
func (c *ZHubClient) Subscribe(topic string, fun func(v string)) {
	c.send("subscribe " + topic)
	if fun != nil {
		c.wlock.Lock()
		defer c.wlock.Unlock()
		c.subFun[topic] = fun
	}
}

func (c *ZHubClient) Unsubscribe(topic string) {
	c.send("unsubscribe " + topic)
	delete(c.subFun, topic)
}

/*
---
ping
---
*/
func (c *ZHubClient) ping() {
	c.send("ping")
}

// Publish -------------------------------------- pub-sub --------------------------------------
func (c *ZHubClient) Publish(topic string, message string) error {
	return c.send("publish", topic, message)
}

func (c *ZHubClient) Broadcast(topic string, message string) error {
	return c.send("broadcast", topic, message)
}

func (c *ZHubClient) Delay(topic string, message string, delay int) error {
	return c.send("delay", topic, message, strconv.Itoa(delay))
}

/*
Timer

	func (c *ZHubClient) Timer(topic string, expr string, fun func()) {
		c.timerFun[topic] = fun
		c.send("timer", topic, expr, "x")
	}
*/
func (c *ZHubClient) Timer(topic string, fun func()) {
	if fun != nil {
		c.timerFun[topic] = fun
	}
	c.send("timer", topic)
}

// Cmd send cmd
func (c *ZHubClient) Cmd(cmd ...string) {
	if len(cmd) == 1 {
		c.send("cmd", cmd[0])
	} else if len(cmd) > 1 {
		cmdx := make([]string, 0)
		cmdx = append(cmdx, "cmd")
		cmdx = append(cmdx, cmd...)
		c.send(cmdx...)
	}
}

func (c *ZHubClient) Close() {
	c.conn.Close()
}

// TryLock
func TryLock(key string, duration int) {
	/*uuid := uuid.New()

	TODO

	return Lock{Key: key, Value: uuid}*/
}

// Lock Key
func (c *ZHubClient) Lock(key string, duration int) Lock {
	uuid := uuid.New()
	c.send("uuid", key, uuid, strconv.Itoa(duration))

	lockChan := make(chan int, 2)
	go func() {
		c.wlock.Lock()
		defer c.wlock.Unlock()
		c.lockFlag[uuid] = &Lock{
			Key:      key,
			Value:    uuid,
			flagChan: lockChan,
		}
	}()

	select {
	case <-lockChan:
		log.Println("lock-ok", time.Now().UnixNano()/1e6, uuid)
	}

	return Lock{Key: key, Value: uuid}
}

func (c *ZHubClient) Unlock(l Lock) {
	c.send("unlock", l.Key, l.Value)
	delete(c.lockFlag, l.Value)
}

// -------------------------------------- rpc --------------------------------------
var rpcMap = make(map[string]*Rpc)
var rpcLock = sync.RWMutex{}

func (c *ZHubClient) rpcInit() {

	// 添加  appname 主题订阅处理
	c.Subscribe(c.appname, func(v string) {
		log.Println("rpc back:", v)
		rpcLock.Lock()
		defer rpcLock.Unlock()

		result := RpcResult{}
		err := json.Unmarshal([]byte(v), &result)
		if err != nil {
			// 返回失败处理
			log.Println("rpc result parse error:", err)
			return
		}

		rpc := rpcMap[result.Ruk]
		if rpc == nil {
			return // 本地已无 rpc 请求等待，如:超时结束
		}

		rpc.RpcResult = result
		close(rpc.Ch) // 发送事件
		delete(rpcMap, result.Ruk)
	})
}

type Rpc struct {
	Ruk   string `json:"ruk"`
	Topic string `json:"topic"`
	Value string `json:"value"`

	Ch        chan int  `json:"-"` //请求返回标记
	RpcResult RpcResult `json:"-"`
}

type RpcResult struct {
	Ruk     string `json:"ruk"`
	Retcode int    `json:"retcode"`
	Retinfo string `json:"retinfo"`
	Result  string `json:"result"`
}

func (r Rpc) backTopic() string {
	return strings.Split(r.Ruk, "::")[0]
}

func (c ZHubClient) Rpc(topic string, message string, back func(res RpcResult)) {
	rpc := Rpc{
		Ruk:   c.appname + "::" + uuid.New(),
		Topic: topic,
		Value: message,
		Ch:    make(chan int, 0),
	}
	bytes, err := json.Marshal(&rpc)
	if err != nil {
		log.Println("rpc marshal error:", err)
	}
	log.Println("rpc call:", string(bytes[:]))
	c.Publish(topic, string(bytes[:]))

	rpcLock.Lock()
	rpcMap[rpc.Ruk] = &rpc
	rpcLock.Unlock()

	select {
	case <-rpc.Ch:
		// ch 事件（rpc 返回）
	case <-time.After(time.Second * 15):
		// rpc 超时
		x, _ := json.Marshal(rpc)
		log.Println("rpc timeout:", x)
		rpc.RpcResult = RpcResult{
			Retcode: 505,
			Retinfo: "请求超时",
		}
	}
	back(rpc.RpcResult)
}

// RpcSubscribe rpc subscribe
func (c ZHubClient) RpcSubscribe(topic string, fun func(Rpc Rpc) RpcResult) {
	c.Subscribe(topic, func(v string) {
		rpc := Rpc{}
		err := json.Unmarshal([]byte(v), &rpc)
		if err != nil {

			return
		}

		result := fun(rpc)
		result.Ruk = rpc.Ruk

		res, _ := json.Marshal(result)
		c.Publish(rpc.backTopic(), string(res))
	})
}

// --------------------------------------------------------------------------------

/*
send socket message :
if len(vs) equal 1 will send message `vs[0] + "\r\n"`
else if len(vs) gt 1 will send message `* + len(vs)+ "\r\n"  +"$"+ len(vs[n])+ "\r\n" + vs[n] + "\r\n" ...`
*/
func (c *ZHubClient) send(vs ...string) (err error) {
	//chSend <- vs
	c.wlock.Lock()
	defer c.wlock.Unlock()
a:
	if len(vs) == 1 {
		_, err = c.conn.Write([]byte(vs[0] + "\r\n"))
	} else if len(vs) > 1 {
		data := "*" + strconv.Itoa(len(vs)) + "\r\n"
		for _, v := range vs {
			data += "$" + strconv.Itoa(utf8.RuneCountInString(v)) + "\r\n"
			data += v + "\r\n"
		}
		_, err = c.conn.Write([]byte(data))
	}
	if err != nil {
		log.Println(err)
		time.Sleep(time.Second * 3)
		// check conn reconnect
		{
			c.wlock.Unlock()
			c.reconn()
			c.wlock.Lock()
		}
		goto a
	}

	return err
}

func (c *ZHubClient) receive() {
	r := bufio.NewReader(c.conn)
	for {
		v, _, err := r.ReadLine()
		if err != nil {
			log.Println(err)
			return
		}
		if len(v) == 0 {
			continue
		}
		switch v[0] {
		case '+':
			if string(v) == "+ping" {
				c.send("+pong")
			}
			log.Println("receive:", string(v))
		case '-':
			log.Println("error:", string(v))
		case '*':
			if len(v) < 2 {
				continue
			}
			n, err := strconv.Atoi(string(v[1:]))
			if err != nil || n <= 0 {
				continue
			}
			var vs []string
			//for i := 0; i < n; i++ {
			for len(vs) < n {
				line, _, err := r.ReadLine()
				if err != nil || line == nil {
					continue
				}
				if len(line) < 2 {
					continue
				}
				clen, err := strconv.Atoi(string(line[1:]))
				if err != nil || clen <= 0 {
					continue
				}
				buf := make([]byte, clen)
				_, err = io.ReadFull(r, buf)
				if err != nil {
					log.Println(err)
					continue
				}
				vs = append(vs, string(buf))
			}
			if len(vs) == 3 && vs[0] == "message" {
				if vs[1] == "lock" {
					go func() {
						log.Println("lock:" + vs[2])
						c.wlock.Lock()
						defer c.wlock.Unlock()
						if c.lockFlag[vs[2]] == nil {
							return
						}
						c.lockFlag[vs[2]].flagChan <- 0
					}()
				} else {
					c.chReceive <- vs
				}
				continue
			}
			if len(vs) == 2 && vs[0] == "timer" {
				c.timerReceive <- vs
				continue
			}
		}
	}
}

// -------------------------------------- hm --------------------------------------

// ==============================================================================

var reconnect = 0

// ClientRun client 命令行程序
func ClientRun(addr string) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s", addr))

	for {
		if err != nil {
			log.Println(err)
			time.Sleep(time.Second * 3)
			conn, err = net.Dial("tcp", fmt.Sprintf("%s", addr))
			continue
		}

		fmt.Println(fmt.Sprintf("had connected server: %s", addr))
		break
	}

	defer func() {
		if reconnect == 1 {
			conn.Close()
			ClientRun(addr)
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
