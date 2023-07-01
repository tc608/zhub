package zsub

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/robfig/cron"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ZTimer struct {
	Conns  []*ZConn
	Expr   string
	Topic  string
	Cron   *cron.Cron
	Ticker *time.Ticker
	Single bool
}

type ZDelay struct {
	Topic    string
	Value    string
	Exectime time.Time
	Timer    *time.Timer
}

// delay topic value 100 -> publish topic value
func (s *ZSub) delay(rcmd []string, c *ZConn) {
	s.Lock()
	defer func() {
		s.Unlock()
		// s.SaveData()
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
			delay.Timer.Stop()
			delete(s.delays, rcmd[1]+"-"+rcmd[2])
			return
		}
		delay.Timer.Reset(time.Duration(t) * time.Millisecond)
	} else {
		if t < 0 {
			return
		}
		delay := &ZDelay{
			Topic:    rcmd[1],
			Value:    rcmd[2],
			Exectime: time.Now().Add(time.Duration(t) * time.Millisecond),
			Timer:    time.NewTimer(time.Duration(t) * time.Millisecond),
		}
		s.delays[rcmd[1]+"-"+rcmd[2]] = delay
		go func() {
			select {
			case <-delay.Timer.C:
				log.Println("delay send:", rcmd[1], rcmd[2])
				Hub.Publish(rcmd[1], rcmd[2])
				funChan <- func() {
					delete(s.delays, rcmd[1]+"-"+rcmd[2])
				}
			}
		}()
	}
}

/*
["Timer", Topic, expr, 0|1]
*/
func (s *ZSub) timer(rcmd []string, c *ZConn) {
	s.Lock()
	defer s.Unlock()
	timer := s.timers[rcmd[1]]
	if timer == nil {
		timer = &ZTimer{
			Conns: []*ZConn{},
			Topic: rcmd[1],
		}
		s.timers[rcmd[1]] = timer
	}
	if c != nil {
		timer.Conns = c.appendTo(timer.Conns)
	}

	if len(rcmd) == 4 && !strings.EqualFold(timer.Expr, rcmd[2]) {
		timer.Expr = rcmd[2]
		if timer.Cron != nil {
			timer.Cron.Stop()
		}
		if timer.Ticker != nil {
			timer.Ticker.Stop()
		}

		var timerFun = func() {
			for _, conn := range timer.Conns {
				log.Println("Timer send:", timer.Topic)
				err := conn.send("Timer", timer.Topic)
				if timer.Single && err == nil {
					break
				}
			}
		}

		r, _ := regexp.Compile("^\\d+[d,H,m,s]$")
		expr := timer.Expr
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

			timer.Ticker = ticker
			go func() {
				for range ticker.C {
					timerFun()
				}
			}()
		} else {
			timer.Cron = func() *cron.Cron {
				c := cron.New()
				c.AddFunc(timer.Expr, timerFun)
				go c.Run()
				return c
			}()
		}
		//Timer.configSave()
	}
	if len(rcmd) == 4 && (strings.EqualFold("a", rcmd[3]) != timer.Single) {
		timer.Single = strings.EqualFold("a", rcmd[3])
		//Timer.configSave()
	}
}

func (s *ZSub) ReloadTimer() {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8",
		Conf.Ztimer.Db.User,
		Conf.Ztimer.Db.Password,
		Conf.Ztimer.Db.Addr,
		Conf.Ztimer.Db.Database,
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
		s.timer([]string{"Timer", name, expr, single}, nil) //["Timer", Topic, expr, a|x]
	}
}
