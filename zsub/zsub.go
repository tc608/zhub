package zsub

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
)

var (
	zsub ZSub = ZSub{
		topics: make(map[string]*ZTopic),
		timers: make(map[string]*ZTimer),
	}
)

type ZSub struct {
	sync.Mutex
	topics map[string]*ZTopic
	timers map[string]*ZTimer
}

type ZConn struct { //ZConn
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
	ztopic := s.topics[topic] //ZTopic
	if ztopic == nil {
		ztopic = &ZTopic{
			groups: map[string]*ZGroup{},
			topic:  topic,
			chMsg:  make(chan string, 100),
		}
		ztopic.init()
		s.topics[topic] = ztopic
	}

	zgroup := ztopic.groups[c.groupid] //ZGroup
	if zgroup == nil {
		zgroup = &ZGroup{
			conns:  []*ZConn{},
			ztopic: ztopic,
			chMsg:  make(chan string, 1000),
		}
		zgroup.init()
		ztopic.groups[c.groupid] = zgroup
	}

	_conns := make([]*ZConn, 0)
	for _, conn := range zgroup.conns {
		if conn == c {
			continue
		}
		_conns = append(_conns, conn)
	}
	_conns = append(_conns, c)
	zgroup.conns = _conns

	// 这是 ZConn
	_topics := c.topics
	for _, _topic := range c.topics {
		if strings.EqualFold(_topic, topic) {
			continue
		}
		_topics = append(_topics, _topic)
	}
	_topics = append(_topics, topic)
	c.topics = _topics
}

/*
取消订阅：
*/
func (s *ZSub) unsubscribe(c *ZConn, topic string) { // 取消订阅 zconn{}
	ztopic := s.topics[topic] //ZTopic
	if ztopic == nil {
		return
	}

	zgroup := ztopic.groups[c.groupid] //ZGroup
	if zgroup == nil {
		return
	}

	_conns := make([]*ZConn, 0)
	for _, conn := range zgroup.conns {
		if conn == c {
			continue
		}
		_conns = append(_conns, c)
	}
	zgroup.conns = _conns
}

/*
发送主题消息
1、写入主题消息列表（_zdb）
2、回复消息写入成功
3、推送主题消息
*/
func (s *ZSub) publish(topic string, msg string) {
	s.Lock()
	defer s.Unlock()
	ztopic := s.topics[topic] //ZTopic
	if ztopic == nil {
		return
	}
	ztopic.chMsg <- msg
	ztopic.mcount++
}

func (s *ZSub) close(c *ZConn) {
	// 订阅
	for _, topic := range c.topics {
		s.unsubscribe(c, topic)
	}

	// 延时

	// timer conn close
	for _, topic := range c.timers { // fixme: 数据逻辑交叉循环
		timer := s.timers[topic]
		if timer != nil {
			timer.close(c)
		}
	}
	(*c.conn).Close()
}

// ==================  ZHub 服务 =====================================
func ServerStart(host string, port int) {
	listen, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Printf("_zdb started listen on: %s:%d \n", host, port)

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
		fmt.Println("conn start: ", conn.RemoteAddr())

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
