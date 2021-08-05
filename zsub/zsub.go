package zsub

import (
	"bufio"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
	"zhub/conf"
)

var (
	zsub = &ZSub{
		topics: make(map[string]*ZTopic),
		timers: make(map[string]*ZTimer),
		delays: make(map[string]*ZDelay),
		locks:  make(map[string][]*Lock),
		conns:  make([]*ZConn, 0),
	}
)

func init() {
	// conn health check: T=10s, close>29s
	go func() {
		ticker := time.NewTicker(time.Second * 10)
		defer ticker.Stop()

		for range ticker.C {
			funChan <- func() {
				defer func() {
					if r := recover(); r != nil {
						log.Println("conn health check Recovered:", r)
					}
				}()
				conns := make([]*ZConn, 0) // 需要关闭的连接
				for _, c := range zsub.conns {
					if c.ping > 0 && c.ping-c.pong > 19 {
						conns = c.appendTo(conns)
						continue
					}

					c.ping = time.Now().Unix()
					if c.pong == 0 {
						c.pong = c.ping
					}

					c.send("+ping")
				}

				// close
				for _, c := range conns {
					log.Println("========================================= conn ping close:", (*c.conn).RemoteAddr(), "[", c.groupid, "] =========================================")
					c.close()
				}

			}
		}
	}()
}

type ZSub struct {
	sync.RWMutex
	topics map[string]*ZTopic
	timers map[string]*ZTimer
	delays map[string]*ZDelay
	locks  map[string][]*Lock
	conns  []*ZConn
}

type ZConn struct { //ZConn
	sync.Mutex
	conn      *net.Conn
	groupid   string
	topics    []string
	timers    []string            // 订阅、定时调度分别创建各自连接
	stoped    chan int            // 关闭信号量
	substoped map[string]chan int // 关闭信号量
	ping      int64               // 最后心跳时间
	pong      int64               // 最后心跳回复时间
}

type Lock struct {
	key      string
	uuid     string
	duration int
	timer    *time.Timer
	start    int64
	//stop     time.Time
}

func NewZConn(conn *net.Conn) *ZConn {
	return &ZConn{
		conn:      conn,
		topics:    []string{},
		timers:    []string{},
		stoped:    make(chan int, 0),
		substoped: make(map[string]chan int),
	}
}

/*
新增订阅：
1、找到对应主题信息
2、加入到对应组别；如果是第一次的消费组 offset从当前 mcount 开始
3、若有待消费消息启动消费
*/
func (c *ZConn) subscribe(topic string) { // 新增订阅 zconn{}
	zsub.Lock()
	defer zsub.Unlock()
	ztopic := zsub.topics[topic] //ZTopic
	if ztopic == nil {
		ztopic = &ZTopic{
			groups: map[string]*ZGroup{},
			topic:  topic,
			chMsg:  make(chan string, 500),
		}
		ztopic.init()
		zsub.topics[topic] = ztopic
	}

	zgroup := ztopic.groups[c.groupid] //ZGroup
	if zgroup == nil {
		zgroup = &ZGroup{
			//conns:  []*ZConn{},
			ztopic: ztopic,
			chMsg:  make(chan string, 500),
		}
		ztopic.groups[c.groupid] = zgroup
	}

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
func (c *ZConn) unsubscribe(topic string) { // 取消订阅 zconn{}
	c.Lock()
	defer c.Unlock()
	close(c.substoped[topic])
	ztopic := zsub.topics[topic] //ZTopic
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

// send message
func (c *ZConn) send(vs ...string) error {
	c.Lock()
	defer c.Unlock()

	var bytes []byte

	if len(vs) == 1 {
		bytes = []byte(vs[0] + "\r\n")
	} else if len(vs) > 1 {
		data := "*" + strconv.Itoa(len(vs)) + "\r\n"
		for _, v := range vs {
			data += "$" + strconv.Itoa(utf8.RuneCountInString(v)) + "\r\n"
			data += v + "\r\n"
		}
		bytes = []byte(data)
	}
	_, err := (*c.conn).Write(bytes)
	return err
}

func (c *ZConn) close() {
	close(c.stoped)
	// sub
	for _, topic := range c.topics {
		c.unsubscribe(topic)
	}

	// timer conn close
	zsub.Lock()
	defer zsub.Unlock()
	for _, topic := range c.timers { // fixme: 数据逻辑交叉循环
		timer := zsub.timers[topic]
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

func (c *ZConn) removeTo(arr []*ZConn) []*ZConn {
	if arr == nil {
		arr = make([]*ZConn, 0)
	}
	for i, item := range arr {
		if item == c {
			arr = append(arr[:i], arr[i+1:]...)
		}
	}
	return arr
}

// ServerStart ==================  ZHub server =====================================
/*
1、初始化服务
2、启动服务监听
*/
func ServerStart(addr string) {
	conf.GetStr("data.dir", "data")

	go func() {
		for {
			fun, ok := <-funChan
			if !ok {
				break
			}
			fun()
		}
	}()

	// 重新加载[定时、延时]
	go zsub.ReloadTimer()
	go zsub.loadDelay()
	//go zsub.loadLock()

	// 启动服务监听
	listen, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("zhub started listen on: %s \n", addr)

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		log.Println("conn start: ", conn.RemoteAddr())

		zConn := NewZConn(&conn)
		go zsub.acceptHandler(zConn)
	}
}

// 连接处理
func (s *ZSub) acceptHandler(c *ZConn) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("acceptHandler Recovered:", r)
		}
	}()
	defer func() {
		// conn remove to conns
		funChan <- func() {
			zsub.conns = c.removeTo(zsub.conns)
		}

		// close ZConn
		c.close()
	}()

	// conn add to conns
	funChan <- func() {
		zsub.conns = c.appendTo(zsub.conns)
	}

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
				line, _, _ := reader.ReadLine()
				clen := 0
				if strings.EqualFold("$", string(line[:1])) {
					clen, _ = strconv.Atoi(string(line[1:]))
				}
				var vx = ""
			a:
				if v, prefix, _ := reader.ReadLine(); prefix {
					vx += string(v)
					goto a
				} else {
					vx += string(v)
				}
				if clen > utf8.RuneCountInString(vx) {
					vx += "\r\n"
					goto a
				}

				rcmd = append(rcmd, vx)
			}
		default:
			rcmd = append(rcmd, string(line))
		}

		if len(rcmd) == 0 {
			continue
		}

		msgAccept(Message{Conn: c, Rcmd: rcmd})
	}
}

/*
accept stop message
1、send message to stop's chan
2、feedback send success to sender, and sending message to stop's subscripts
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
	if strings.EqualFold(topic, "lock") {
		log.Println("lock", msg)
	}

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

/*
lock: 	lock   key uuid t
unlock: unlock key uuid
*/
func (s *ZSub) _lock(lock *Lock) {
	locks := s.locks[lock.key]
	if locks == nil {
		locks = make([]*Lock, 0)
	}
	if len(locks) == 0 { // lock success
		lock.start = time.Now().Unix()
		locks = append(locks, lock)
		s.locks[lock.key] = locks
		s.broadcast("lock", lock.uuid)

		// 设置时间到解锁
		locks[0].timer = time.NewTimer(time.Duration(locks[0].duration) * time.Second)
		go func() {
			select {
			case <-locks[0].timer.C:
				s._unlock(*locks[0])
			}
		}()
	} else {
		s.locks[lock.key] = append(locks, lock)
	}
}
func (s *ZSub) _unlock(l Lock) {
	locks := s.locks[l.key]
	if locks == nil || len(locks) == 0 {
		return
	}
	if strings.EqualFold(locks[0].uuid, l.uuid) {
		locks[0].timer.Stop()
		locks = locks[1:]
		s.locks[l.key] = locks
	}
	if len(s.locks[l.key]) > 0 { // next lock
		s.broadcast("lock", s.locks[l.key][0].uuid)
		s.locks[l.key][0].start = time.Now().Unix()
		s.locks[l.key][0].timer = time.NewTimer(time.Duration(s.locks[l.key][0].duration) * time.Second)
		go func() {
			select {
			case <-s.locks[l.key][0].timer.C:
				s._unlock(*s.locks[l.key][0])
			}
		}()
	}
}

func (s *ZSub) shutdown() {
	s.dataStorage()
	os.Exit(0)
}

func Info() map[string]interface{} {
	m := map[string]interface{}{}

	for s, topic := range zsub.topics {
		// {groups:[{name:xxx,size:xx}]}
		arr := make([]map[string]interface{}, 0)

		for groupname, group := range topic.groups {
			arr = append(arr, map[string]interface{}{
				"name":    groupname,
				"subsize": len(group.conns),
				"offset":  group.offset,
				"mcount":  topic.mcount,
			})
		}
		m[s] = arr
	}

	return m
}

func (s *ZSub) Clearup() {
	for tn, topic := range s.topics {
		for _, group := range topic.groups {
			if len(group.conns) > 0 || topic.mcount > group.offset {
				goto a
			}
		}
		close(topic.chMsg)
		delete(s.topics, tn)
	a:
	}
}

func ZSubx() *ZSub {
	return zsub
}
