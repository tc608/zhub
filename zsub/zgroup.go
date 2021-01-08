package zsub

import "sync"

type ZGroup struct { //ZGroup
	sync.Mutex
	conns  []*ZConn
	offset int
	chMsg  chan string // 组消息即时投递
}

func createZGroup(c *ZConn) *ZGroup {
	zgroup := &ZGroup{
		conns: []*ZConn{},
		chMsg: make(chan string, 100),
	}

	// 开启消息推送
	go func() {
		for {
			msg, ok := <-zgroup.chMsg
			if !ok {
				break
			}

			for _, c := range zgroup.conns {
				(*c.conn).Write([]byte(msg))
				zgroup.offset++
			}
		}
	}()
	return zgroup
}
