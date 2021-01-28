package zsub

import (
	"bufio"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
)

var (
	zsub = ZSub{
		topics: make(map[string]*ZTopic),
		timers: make(map[string]*ZTimer),
	}
)

type ZSub struct {
	sync.RWMutex
	topics map[string]*ZTopic
	timers map[string]*ZTimer
}

type ZConn struct { //ZConn
	sync.Mutex
	conn    *net.Conn
	groupid string
	topics  []string
	timers  []string // 订阅、定时调度分别创建各自连接
}

/*
新增订阅：
1、找到对应主题信息
2、加入到对应组别；如果是第一次的消费组 offset从当前 mcount 开始
3、若有待消费消息启动消费
*/
func (s *ZSub) subscribe(c *ZConn, topic string) { // 新增订阅 zconn{}
	s.Lock()
	defer s.Unlock()
	ztopic := s.topics[topic] //ZTopic
	if ztopic == nil {
		ztopic = &ZTopic{
			groups: map[string]*ZGroup{},
			topic:  topic,
			chMsg:  make(chan string, 10000),
		}
		ztopic.init()
		s.topics[topic] = ztopic
	}

	zgroup := ztopic.groups[c.groupid] //ZGroup
	if zgroup == nil {
		zgroup = &ZGroup{
			//conns:  []*ZConn{},
			ztopic: ztopic,
			chMsg:  make(chan string, 1000),
		}
		ztopic.groups[c.groupid] = zgroup
	}

	//zgroup.conns = c.appendTo(zgroup.conns)
	zgroup.appendTo(c)

	for i, item := range c.topics {
		if strings.EqualFold(item, topic) {
			c.topics = append(c.topics[:i], c.topics[i+1:]...)
		}
	}
	c.topics = append(c.topics, topic)
}

/*
取消订阅：
*/
func (s *ZSub) unsubscribe(c *ZConn, topic string) { // 取消订阅 zconn{}
	s.Lock()
	defer s.Unlock()
	ztopic := s.topics[topic] //ZTopic
	if ztopic == nil {
		return
	}

	zgroup := ztopic.groups[c.groupid] //ZGroup
	if zgroup == nil {
		return
	}

	for i, item := range zgroup.conns {
		if item == c {
			zgroup.conns = append(zgroup.conns[:i], zgroup.conns[i+1:]...)
		}
	}
}

/*
accept topic message
1、send message to topic's chan
2、feedback send success to sender, and sending message to topic's subscripts
*/
func (s *ZSub) publish(topic, msg string) {
	s.RLock()
	defer s.RUnlock()
	ztopic := s.topics[topic] //ZTopic
	if ztopic == nil {
		return
	}
	ztopic.chMsg <- msg
	ztopic.mcount++
}

/*
send broadcast message
*/
func (s *ZSub) broadcast(topic, msg string) {
	s.RLock()
	defer s.RUnlock()

	ztopic := s.topics[topic] //ZTopic
	if ztopic == nil {
		return
	}

	for _, group := range ztopic.groups {
		for _, conn := range group.conns {
			conn.send("message", topic, msg)
		}
	}
}

func (s *ZSub) close(c *ZConn) {
	// sub
	for _, topic := range c.topics {
		s.unsubscribe(c, topic)
	}

	// daly

	// timer conn close
	s.Lock()
	defer s.Unlock()
	for _, topic := range c.timers { // fixme: 数据逻辑交叉循环
		timer := s.timers[topic]
		if timer == nil {
			continue
		}

		for i, item := range timer.conns {
			if item == c {
				timer.conns = append(timer.conns[:i], timer.conns[i+1:]...)
			}
		}
	}
	(*c.conn).Close()
}

func (c *ZConn) appendTo(arr []*ZConn) []*ZConn {
	if arr == nil {
		arr = make([]*ZConn, 0)
	}
	for i, item := range arr {
		if item == c {
			arr = append(arr[:i], arr[i+1:]...)
		}
	}
	return append(arr, c)
}

// ==================  ZHub server =====================================
/*
1、初始化服务
2、启动服务监听
*/
func ServerStart(addr string) {

	// 加载定时调度服务
	zsub.reloadTimerConfig()

	// 启动服务监听
	listen, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("zhub started listen on: %s \n", addr)

	// 启动消息监听处理
	go func() {
		for {
			v, ok := <-chanMessages
			if !ok {
				break
			}

			// 事件消费
			msgAccept(v)
		}
	}()

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		log.Println("conn start: ", conn.RemoteAddr())

		go zsub.acceptHandler(&ZConn{
			conn:   &conn,
			topics: []string{},
			timers: []string{},
		})
	}
}

// 连接处理
func (s *ZSub) acceptHandler(c *ZConn) {
	defer func() {
		s.close(c) // close ZConn
	}()

	reader := bufio.NewReader(*c.conn)
	for {
		rcmd := make([]string, 0)
		line, _, err := reader.ReadLine()
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

		// 接收消息 zdb fixme： 细节暴露太多
		chanMessages <- Message{Conn: c, Rcmd: rcmd}
	}
}
