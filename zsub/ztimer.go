package zsub

import (
	"bytes"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/robfig/cron"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

type ZTimer struct {
	conns  []*ZConn
	expr   string
	topic  string
	cron   *cron.Cron
	ticker *time.Ticker
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
	if c != nil {
		timer.conns = c.appendTo(timer.conns)
	}

	if len(rcmd) == 4 && !strings.EqualFold(timer.expr, rcmd[2]) {
		timer.expr = rcmd[2]
		if timer.cron != nil {
			timer.cron.Stop()
		}
		if timer.ticker != nil {
			timer.ticker.Stop()
		}

		var timerFun = func() {
			for _, conn := range timer.conns {
				log.Println("timer send:", rcmd[1], rcmd[2])
				err := conn.send("timer", timer.topic)
				if timer.single && err == nil {
					break
				}
			}
		}

		r, _ := regexp.Compile("^\\d+[d,H,m,s]$")
		expr := timer.expr
		if r.MatchString(expr) {
			n, _ := strconv.Atoi(expr[:len(expr)-1])
			_n := time.Duration(n)
			var ticker *time.Ticker
			switch expr[len(expr)-1:] {
			case "d":
				ticker = time.NewTicker(_n * time.Hour * 24)
			case "H":
				ticker = time.NewTicker(_n * time.Hour)
			case "m":
				ticker = time.NewTicker(_n * time.Minute)
			case "s":
				ticker = time.NewTicker(_n * time.Second)
			}

			timer.ticker = ticker
			go func() {
				for range ticker.C {
					timerFun()
				}
			}()
		} else {
			timer.cron = func() *cron.Cron {
				c := cron.New()
				c.AddFunc(timer.expr, timerFun)
				go c.Run()
				return c
			}()
		}
		//timer.configSave()
	}
	if len(rcmd) == 4 && (strings.EqualFold("a", rcmd[3]) != timer.single) {
		timer.single = strings.EqualFold("a", rcmd[3])
		//timer.configSave()
	}
}

/*func (t *ZTimer) close(c *ZConn) {
	for i, item := range t.conns {
		if item.conn == c.conn {
			t.conns = append(t.conns[:i], t.conns[i+1:]...)
		}
	}
}*/

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

	//fmt.Println(buf.String())

	rest, err, s := executeShell(buf.String())
	if err != nil {
		log.Println(err)
	}
	if !strings.EqualFold(rest, "") {
		fmt.Println("res:", rest)
	}
	if !strings.EqualFold(s, "") {
		fmt.Println("error-rest:", s)
	}
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

func (s *ZSub) ReloadTimer() {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8",
		GetStr("ztimer.db.user", "root"),
		GetStr("ztimer.db.pwd", "123456"),
		GetStr("ztimer.db.addr", "127.0.0.1:3306"),
		GetStr("ztimer.db.database", "zhub"),
	))

	if err != nil {
		log.Println(err)
		return
	}
	defer db.Close()
	rows, err := db.Query("SELECT t.`name`, IF(t.`status`=10,t.`expr`,''), IF(t.`single`=1,'a','x') 'single' FROM tasktimer t ORDER BY t.`timerid`")
	if err != nil {
		log.Println(err)
		return
	}

	for rows.Next() {
		var name string
		var expr string
		var single string
		rows.Scan(&name, &expr, &single)
		s.timer([]string{"timer", name, expr, single}, nil) //["timer", topic, expr, a|x]
	}
}

// ==================  delay =====================================

type ZDelay struct {
	topic    string
	value    string
	exectime time.Time
	timer    *time.Timer
}

// delay topic value 100 -> publish topic value
func (s *ZSub) delay(rcmd []string, c *ZConn) {
	s.Lock()
	defer func() {
		s.Unlock()
		// s.dataStorage()
		s.delayup = true
	}()
	if len(rcmd) != 4 {
		c.send("-Error: subscribe para number!")
		return
	}

	t, err := strconv.ParseInt(rcmd[3], 10, 64)
	if err != nil {
		c.send("-Error: " + strings.Join(rcmd, " "))
		return
	}

	delay := s.delays[rcmd[1]+"-"+rcmd[2]]
	if delay != nil {
		if t < 0 {
			delay.timer.Stop()
			delete(s.delays, rcmd[1]+"-"+rcmd[2])
			return
		}
		delay.timer.Reset(time.Duration(t) * time.Millisecond)
	} else {
		if t < 0 {
			return
		}
		delay := &ZDelay{
			topic:    rcmd[1],
			value:    rcmd[2],
			exectime: time.Now().Add(time.Duration(t) * time.Millisecond),
			timer:    time.NewTimer(time.Duration(t) * time.Millisecond),
		}
		s.delays[rcmd[1]+"-"+rcmd[2]] = delay
		go func() {
			select {
			case <-delay.timer.C:
				log.Println("delay send:", rcmd[1], rcmd[2])
				zsub.Publish(rcmd[1], rcmd[2])
				funChan <- func() {
					delete(s.delays, rcmd[1]+"-"+rcmd[2])
				}
			}
		}()
	}
}
