package zsub

import "sync"

type ZGroup struct { // ZGroup
	sync.Mutex
	conns  []*ZConn
	offset int
	chMsg  chan string // 组消息即时投递
	ztopic *ZTopic     // 所属topic
}

func (g *ZGroup) init() {
	go func() {
		for {
			msg, ok := <-g.chMsg
			if !ok {
				break
			}

			if len(g.conns) == 0 {
				continue
			}
			g.conns[0].send("message", g.ztopic.topic, msg)
			g.offset++
		}
	}()
}
