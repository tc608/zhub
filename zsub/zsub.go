package zsub

import (
	"net"
	"sync"
)

var (
	zsub ZSub
)

type ZSub struct {
	sync.Mutex
	topics map[string]*ZTopic
}


type ZConn struct { //ZConn
	conn    *net.Conn
	groupid string
	topics  []string
}

/*
新增订阅：
1、找到对应主题信息
2、加入到对应组别；如果是第一次的消费组 offset从当前 mcount 开始
3、若有待消费消息启动消费
*/
func (s ZSub) subscribe(c *ZConn, topic string) { // 新增订阅 zconn{}
	ztopic := s.topics[topic] //ZTopic
	if ztopic == nil {
		ztopic = &ZTopic{groups: map[string]*ZGroup{}}
		s.topics[topic] = ztopic
	}

	zgroup := ztopic.groups[c.groupid] //ZGroup
	if zgroup == nil {
		zgroup = &ZGroup{conns: []*ZConn{}}
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
}

/*
取消订阅：
*/
func (s ZSub) unsubscribe(c *ZConn, topic string) { // 取消订阅 zconn{}
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
1、写入主题消息列表（zdb）
2、回复消息写入成功
3、推送主题消息
*/
func (s ZSub) publish(topic string, message string) {
	s.Lock()
	defer s.Unlock()
	ztopic := s.topics[topic] //ZTopic
	if ztopic == nil {
		return
	}

	for _, zgroup := range ztopic.groups {
		zgroup.chMsg <- message
	}
}
