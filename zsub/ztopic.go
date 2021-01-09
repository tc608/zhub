package zsub

import "sync"

type ZTopic struct { //ZTopic
	sync.Mutex
	groups map[string]*ZGroup
	mcount int
	chMsg  chan string // 主题消息投递
}

// 主题消息发送

//
