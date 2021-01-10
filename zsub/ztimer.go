package zsub

import (
	"fmt"
	"github.com/robfig/cron"
	"strings"
)

type ZTimer struct {
	conns []*ZConn
	expr  string
	topic string
	cron  *cron.Cron
}

func (s *ZSub) timer(rcmd []string, c *ZConn) {
	timer := s.timers[rcmd[1]]
	if timer == nil {
		timer = &ZTimer{
			conns: []*ZConn{},
			topic: rcmd[1],
		}
		s.timers[rcmd[1]] = timer
	}

	_conns := make([]*ZConn, 0)
	for _, conn := range timer.conns {
		if conn == c {
			continue
		}
		_conns = append(_conns, conn)
	}
	_conns = append(_conns, c)
	timer.conns = _conns

	if !strings.EqualFold(timer.expr, rcmd[2]) {
		timer.expr = rcmd[2]
		if timer.cron != nil {
			timer.cron.Stop()
		}
		timer.cron = func() *cron.Cron {
			c := cron.New()
			c.AddFunc(timer.expr, func() {
				//fmt.Println(time.Now().Second())
				for _, conn := range timer.conns {
					send(conn.conn, "timer", timer.topic)
				}
			})
			go c.Run()
			return c
		}()
	}

	s.timers[rcmd[1]] = timer
	fmt.Println("xx")
}

func (t *ZTimer) close(c *ZConn) {
	for i, conn := range t.conns {
		if conn.conn == c.conn {
			t.conns = append(t.conns[:i], t.conns[i+1:]...)
		}
	}
	t.conns = append(t.conns, c)
}
