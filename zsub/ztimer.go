package zsub

import (
	"bytes"
	"fmt"
	"github.com/robfig/cron"
	"log"
	"os/exec"
	"strings"
	"text/template"
)

type ZTimer struct {
	conns  []*ZConn
	expr   string
	topic  string
	cron   *cron.Cron
	single bool
}

/*
["timer", topic, expr, a|x]
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
	if len(rcmd) == 4 && !strings.EqualFold(timer.expr, rcmd[2]) {
		timer.expr = rcmd[2]
		if timer.cron != nil {
			timer.cron.Stop()
		}
		timer.cron = func() *cron.Cron {
			c := cron.New()
			c.AddFunc(timer.expr, func() {
				for _, conn := range timer.conns {
					err := send(conn.conn, "timer", timer.topic)
					if timer.single && err == nil {
						break
					}
				}
			})
			go c.Run()
			return c
		}()
		timer.configSave()
	}
	if len(rcmd) == 4 && !strings.EqualFold("a", rcmd[3]) && !timer.single {
		timer.single = true
		timer.configSave()
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

func (t *ZTimer) configSave() {
	tpl, err := template.New("").Parse(`
	if [ ! -d "/etc/zhub" ]; then
	  mkdir /etc/zhub
	fi
	if [ ! -f "/etc/zhub/ztimer.cron" ]; then
	  touch /etc/zhub/ztimer.cron
	fi

	sed -i /^{{.Name}}\|*/d /etc/zhub/ztimer.cron
	echo '{{.Name}}|{{.Expr}}|{{.Single}}' >> /etc/zhub/ztimer.cron
	`)

	if err != nil {
		log.Println(err)
	}

	var buf bytes.Buffer
	err = tpl.Execute(&buf, map[string]string{
		"Name": t.topic,
		"Expr": t.expr,
		"Single": func() string {
			if t.single {
				return "a"
			} else {
				return "x"
			}
		}(),
	})
	if err != nil {
		log.Println(err)
	}

	fmt.Println(buf.String())

	rest, err, s := executeShell(buf.String())
	if err != nil {
		log.Println(err)
	}
	fmt.Println("res:", rest)
	fmt.Println("error-rest:", s)
}

func executeShell(command string) (string, error, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("/bin/bash", "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), err, stderr.String()
}
