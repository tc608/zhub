package _zdb

import (
	"fmt"
	"github.com/robfig/cron"
	"net"
	"strings"
	"time"
)

var (
	zTimer = make(map[string]*ZTimer)
)

type ZTimer struct {
	conns []*net.Conn
	expr  string
	topic string
	cron  *cron.Cron
}

func timer(rcmd []string, conn net.Conn) {
	ztimer := zTimer[rcmd[1]]
	if ztimer == nil {
		ztimer = &ZTimer{
			conns: []*net.Conn{},
			topic: rcmd[1],
		}
		zTimer[rcmd[1]] = ztimer
	}

	_conns := make([]*net.Conn, 0)
	for _, c := range ztimer.conns {
		if *&conn == *c {
			continue
		}
		_conns = append(_conns, c)
	}
	_conns = append(_conns, &conn)
	ztimer.conns = _conns

	if !strings.EqualFold(ztimer.expr, rcmd[2]) {
		ztimer.expr = rcmd[2]
		if ztimer.cron != nil {
			ztimer.cron.Stop()
		}
		ztimer.cron = func() *cron.Cron {
			c := cron.New()
			c.AddFunc(ztimer.expr, func() {
				fmt.Println(time.Now().Second())
				for _, conn := range ztimer.conns {
					Send(*conn, "timer", ztimer.topic)
				}
			})
			go c.Run()
			return c
		}()
	}

	zTimer[ztimer.topic] = ztimer
	fmt.Println("xx")
}
