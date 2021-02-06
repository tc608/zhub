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
	topic := g.ztopic.topic

	// report subscribe topic check
	if c.substoped[topic] != nil {
		return
	}

	// create new goroutine consumer message
	c.substoped[topic] = make(chan int, 0)
	c.appendTo(g.conns)
	go func() {
		for {
			select {
			case msg, ok := <-g.chMsg:
				if !ok {
					return
				}

				err := c.send("message", topic, msg)
				if err != nil { // 失败处理
					g.chMsg <- msg
					return
				}
				atomic.AddInt32(&g.offset, 1)
			case <-c.stoped:
				return
			case <-c.substoped[topic]:
				delete(c.substoped, topic)
				return
			}
		}
	}()
}
