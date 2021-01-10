package zsub

import (
	"github.com/robfig/cron"
	"strings"
)

type ZTimer struct {
	conns []*ZConn
	expr  string
	topic string
	cron  *cron.Cron
}

/*
1、["timer", topic, expr]
2、["timer", topic]
*/
func (s *ZSub) timer(rcmd []string, c *ZConn) {
	s.Lock()
	defer s.Unlock()
	timer := s.timers[rcmd[1]]
	if timer == nil {
		timer = &ZTimer{
			conns: []*ZConn{},
			topic: rcmd[1],
		}
		s.timers[rcmd[1]] = timer
	}
	timer.conns = c.appendTo(timer.conns)

	// todo: when timer.expr changed send message to all the timer‘s subscribe
	if len(rcmd) == 3 && !strings.EqualFold(timer.expr, rcmd[2]) {
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
}

func (t *ZTimer) close(c *ZConn) {
	for i, item := range t.conns {
		if item.conn == c.conn {
			t.conns = append(t.conns[:i], t.conns[i+1:]...)
		}
	}
}
