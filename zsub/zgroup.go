package zsub

import (
	"sync"
	"sync/atomic"
)

type ZGroup struct { // ZGroup
	sync.Mutex
	conns  []*ZConn
	offset int32
	chMsg  chan string // 组消息即时投递
	ztopic *ZTopic     // 所属topic
}

func (g *ZGroup) appendTo(c *ZConn) {
	c.appendTo(g.conns)
	go func() { // 每个连接开启一个携程发送数据
		for {
			select {
			case msg, ok := <-g.chMsg:
				if !ok {
					return
				}

				err := c.send("message", g.ztopic.topic, msg)
				if err != nil { // 失败处理
					g.chMsg <- msg
					return
				}
				atomic.AddInt32(&g.offset, 1)
			case <-c.stoped:
				return
			}
		}
	}()
}
