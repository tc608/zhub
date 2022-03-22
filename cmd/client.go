package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/go-basic/uuid"
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

type Client struct {
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
}

type Lock struct {
	Key      string   // lock Key
	Uuid     string   // lock Uuid
	flagChan chan int //
	// starttime uint32 // lock start time
	// duration  int    // lock duration
}

func Create(appname string, addr string, groupid string) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return &Client{}, err
	}

	client := Client{
		wlock:      sync.Mutex{},
		rlock:      sync.Mutex{},
		appname:    appname,
		addr:       addr,
		conn:       conn,
		groupid:    groupid,
		createTime: time.Now(),

		subFun:       make(map[string]func(v string)),
		timerFun:     make(map[string]func()),
		chSend:       make(chan []string, 100),
		chReceive:    make(chan []string, 100),
		timerReceive: make(chan []string, 100),
		lockFlag:     make(map[string]*Lock),
	}

	client.send("groupid " + groupid)
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
			c.send("groupid " + c.groupid)
			go c.receive()

			// 重新订阅
			for topic, _ := range c.subFun {
				c.Subscribe(topic, nil)
			}
			for topic, _ := range c.timerFun {
				c.Timer(topic, nil)
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
	c.rpcInit()

	go c.receive()
}

/*
// subscribe topic
---
subscribe x y z
---
*/
func (c *Client) Subscribe(topic string, fun func(v string)) {
	c.send("subscribe " + topic)
	if fun != nil {
		c.wlock.Lock()
		defer c.wlock.Unlock()
		c.subFun[topic] = fun
	}
}

func (c *Client) Unsubscribe(topic string) {
	c.send("unsubscribe " + topic)
	delete(c.subFun, topic)
}

/*
---
ping
---
*/
func (c *Client) ping() {
	c.send("ping")
}

//Publish -------------------------------------- pub-sub --------------------------------------
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
	return c.send("publish", topic, message)
}

func (c *Client) Broadcast(topic string, message string) error {
	return c.send("broadcast", topic, message)
}

func (c *Client) Delay(topic string, message string, delay int) error {
	return c.send("delay", topic, message, strconv.Itoa(delay))
}

/*
Timer
func (c *Client) Timer(topic string, expr string, fun func()) {
	c.timerFun[topic] = fun
	c.send("timer", topic, expr, "x")
}*/
func (c *Client) Timer(topic string, fun func()) {
	if fun != nil {
		c.timerFun[topic] = fun
	}
	c.send("timer", topic)
}

// send cmd
func (c *Client) Cmd(cmd ...string) {
	if len(cmd) == 1 {
		c.send("cmd", cmd[0])
	} else if len(cmd) > 1 {
		cmdx := make([]string, 0)
		cmdx = append(cmdx, "cmd")
		cmdx = append(cmdx, cmd...)
		c.send(cmdx...)
	}
}

func (c *Client) Close() {
	c.conn.Close()
}

// Lock Key
func (c *Client) Lock(key string, duration int) Lock {
	uuid := uuid.New()
	c.send("lock", key, uuid, strconv.Itoa(duration))

	lockChan := make(chan int, 2)
	go func() {
		c.wlock.Lock()
		defer c.wlock.Unlock()
		c.lockFlag[uuid] = &Lock{
			Key:      key,
			Uuid:     uuid,
			flagChan: lockChan,
		}
	}()

	select {
	case <-lockChan:
		log.Println("lock-ok", time.Now().UnixNano()/1e6, uuid)
	}

	return Lock{Key: key, Uuid: uuid}
}

func (c *Client) Unlock(l Lock) {
	c.send("unlock", l.Key, l.Uuid)
	delete(c.lockFlag, l.Uuid)
}

// -------------------------------------- rpc --------------------------------------
var rpcMap = make(map[string]*Rpc)
var rpcLock = sync.RWMutex{}

func (c *Client) rpcInit() {

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

func (c Client) Rpc(topic string, message string, back func(res RpcResult)) {
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

// rpc subscribe
func (c Client) RpcSubscribe(topic string, fun func(Rpc Rpc) RpcResult) {
	c.Subscribe(topic, func(v string) {
		rpc := Rpc{}
		err := json.Unmarshal([]byte(v), &rpc)
		if err != nil {

			return
		}

		result := fun(rpc)
		result.Ruk = rpc.Ruk

		res, _ := json.Marshal(result)
		c.Publish(rpc.backTopic(), string(res[:]))
	})
}

// --------------------------------------------------------------------------------

/*func (c *Client) subscribes(topics ...string) error {
	if len(topics) == 0 {
		return nil
	}

	messages := "subscribe"
	for _, topic := range topics {
		messages += " " + topic
	}
	c.send(messages)
	return nil
}*/

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
			data += "$" + strconv.Itoa(utf8.RuneCountInString(v)) + "\r\n"
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
				if strings.EqualFold(vs[1], "lock") { // message lock Uuid
					go func() {
						log.Println("lock:" + vs[2])
						c.wlock.Lock()
						defer c.wlock.Unlock()

						if c.lockFlag[vs[2]] == nil {
							return
						}
						c.lockFlag[vs[2]].flagChan <- 0
					}()
					continue
				}
				c.chReceive <- vs
				continue
			}
			if len(vs) == 2 && strings.EqualFold(vs[0], "timer") {
				c.timerReceive <- vs
				continue
			}
			/*if len(vs) == 2 && strings.EqualFold(vs[0], "delay") {
				c.delayFun[vs[1]]()
				delete(c.delayFun, vs[1])
				continue
			}*/

			continue
		case "+": // +pong, +xxx
			if strings.EqualFold("+ping", string(v)) { // 心跳消息回复
				c.send("+pong")
			}
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
