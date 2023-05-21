package zsub

import (
	"fmt"
	"log"
	"sync"
)

type ZTopic struct { //ZTopic
	sync.Mutex
	groups map[string]*ZGroup
	mcount int32
	topic  string      // 主题名称
	chMsg  chan string // 主题消息投递
}

// 主题消息发送
func (t *ZTopic) init() {
	go func() {
		for {
			msg, ok := <-t.chMsg
			if !ok {
				break
			}

			for groupName, group := range t.groups {
				// zgroup chan overload check
				if len(group.chMsg) == cap(group.chMsg) {
					log.Println(fmt.Sprintf("zgroup no cap: [%s.%s %s]", groupName, t.topic, msg))
					continue
				}
				group.chMsg <- msg
			}
		}
	}()
}

//
