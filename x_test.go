package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"zhub/cli"
)

type LogInfo2 struct {
	RemoteAddr    string `json:"remote_addr"`            // IP
	Time          string `json:"time"`                   // 请求时间
	Status        string `json:"status"`                 // 请求状态
	BodyBytesSent string `json:"body_bytes_sent"`        // 返回内容字节
	Host          string `json:"host"`                   // 请求域名
	HttpUserAgent string `json:"http_user_agent"`        // 客户端信息
	CostTime      string `json:"upstream_response_time"` // 耗时

	Request         string `json:"request"`          //
	HttpMethod      string `json:"http_method"`      // 请求类型
	Uri             string `json:"uri"`              // uri
	ProtocolVersion string `json:"protocol_version"` // 请求
	RequestTime     string `json:"request_time"`     // 时间戳
	HttpCookie	string `json:"http_cookie"`
}

func TestName(t *testing.T) {
	//client, err := cli.Create("39.108.56.246:1216", "")
	client, err := cli.Create("127.0.0.1:1216", "")
	if err != nil {
		log.Fatal(err)
	}
	mutex := sync.Mutex{}
	n := 0
	//client.Init()
	client.Subscribe("pro-nginx-log", func(v string) {
		mutex.Lock()
		defer mutex.Unlock()

		if strings.Index(v, "api-oss.woaihaoyouxi.com") > -1 {
			return
		}
		if strings.Index(v, "kibana.woaihaoyouxi.com") > -1 {
			return
		}

		/*if strings.Index(v, "ef737b680be2cf7868cca99101fa7e66") == -1 {
			return
		}*/

		n++
		info, err := logParse(v)
		if err != nil {
			log.Println("json parse error", v)
			return
		}
		//fmt.Println(strconv.Itoa(n), "接收到主题 pro-nginx-log 消息", v)
		t, err := strconv.ParseInt(info.RequestTime, 10, 0)
		fmt.Println(strconv.Itoa(n), time.Unix(t, 0).Format("2006-01-02 15:04:05"), info.Status, info.CostTime, info.Uri, info.HttpCookie)
	})

	client.Subscribe("a-1", func(v string) {
		log.Println(v)
	})

	client.Timer("t", "*/3 * * * * *")

	go func() {
		for i := 0; i < 50000; i++ {
			client.Publish("a-1", strconv.Itoa(i))
			time.Sleep(time.Second)
		}
	}()

	//log.Println("send")
	//client.Daly("x", "abx", 1000 * 10)
	time.Sleep(time.Hour * 3)
}

func TestX(t *testing.T) {
	strs := [...]string{"1", "2", "3", "4"}

	strss := strs[0:2]

	fmt.Println(strss)
}

func logParse(str string) (LogInfo2, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("nginx.logParse error:", r, str)
		}
	}()

	var info LogInfo2
	err := json.Unmarshal([]byte(str), &info)
	if err != nil {
		log.Println("111", err, str)
		return LogInfo2{}, err
	}

	if !strings.EqualFold(info.Request, "") {
		arr := strings.Split(info.Request, " ")
		if len(arr) == 3 {
			info.HttpMethod = arr[0]
			info.Uri = arr[1]
			info.ProtocolVersion = arr[2]
		}
	}
	if !strings.EqualFold(info.Request, "") {
		arr := strings.Split(info.Request, " ")
		if len(arr) == 3 {
			info.HttpMethod = arr[0]
			info.Uri = arr[1]
			info.ProtocolVersion = arr[2]
		}
	}

	if !strings.EqualFold(info.Time, "") {
		t, err := time.Parse(time.RFC3339, info.Time)
		if err != nil {
			log.Println("127", err, str)
			return LogInfo2{}, err
		} else {
			info.RequestTime = strconv.FormatInt(t.Unix(), 10)
		}
	}

	return info, nil
}
