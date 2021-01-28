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
			msg, ok := <-g.chMsg
			if !ok {
				break
			}

			err := c.send("message", g.ztopic.topic, msg)
			if err != nil { // 失败处理
				g.chMsg <- msg
				break
			}
			atomic.AddInt32(&g.offset, 1)
		}
	}()
}
