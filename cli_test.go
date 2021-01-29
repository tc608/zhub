package main

import (
	"log"
	"strconv"
	"testing"
	"time"
	"zhub/cli"
)

var (
	//addr = "47.111.150.118:6066"
	addr = "127.0.0.1:1216"
	//addr = "122.112.180.156:6066"
	//addr = "39.108.56.246:1216"
)

func TestCli(t *testing.T) {
	//client, err := cli.Create("39.108.56.246:7070", "")
	client, err := cli.Create(addr, "xx")
	//client, err := cli.Create(addr, "topic-x")
	if err != nil {
		log.Fatal(err)
	}

	// 订阅主题 消息
	client.Subscribe("ax", func(v string) {
		log.Println("收到主题 ax 消息 " + v)
	})

	// 定时调度
	client.Timer("a", func() {
		log.Println("收到 a 定时消息")
	})

	client.Subscribe("a", func(v string) {
		log.Println("收到主题 a 消息 " + v)
	})
	client.Daly("a", "x", 3000)

	time.Sleep(time.Hour * 3)
}

func TestTimer(t *testing.T) {
	go func() {
		client, _ := cli.Create(addr, "topic-1")

		client.Subscribe("ax1", func(v string) {
			log.Println("topic-1-ax: " + v)
		})
	}()
	go func() {
		client, _ := cli.Create(addr, "topic-1")

		client.Subscribe("ax1", func(v string) {
			log.Println("topic-2-ax: " + v)
		})
	}()

	go func() {
		client, _ := cli.Create(addr, "topic-1")

		client.Subscribe("ax1", func(v string) {
			log.Println("topic-3-ax: " + v)
		})
	}()

	/*go func() {
		client, _ := cli.Create(addr, "topic-1")

		client.Subscribe("ax", func(v string) {
			log.Println("topic-4-ax: " + v)
		})
	}()
	go func() {
		client, _ := cli.Create(addr, "topic-1")

		client.Subscribe("ax", func(v string) {
			log.Println("topic-5-ax: " + v)
		})
	}()*/

	/*go func() {
		client, _ := cli.Create(addr, "topic-2")
		client.Timer("a", func() {
			log.Println("client-2 收到 a 的定时消息")
		})
	}()

	go func() {
		client, _ := cli.Create(addr, "topic-3")
		client.Timer("c", func() {
			log.Println("client-2 收到 c 的定时消息")
		})

		client.Timer("b", func() {
			log.Println("client-2 收到 b 的定时消息")
		})
		client.Timer("LOAD-LIVE-ROOM-UNBANNED", func() {
			log.Println("client-2 收到 LOAD-LIVE-ROOM-UNBANNED 的定时消息")
		})
		client.Timer("VIP-EXP-EXPIRE", func() {
			log.Println("client-2 收到 VIP-EXP-EXPIRE 的定时消息")
		})
	}()*/

	time.Sleep(time.Hour * 3)
}

func TestSendCmd(t *testing.T) {
	client, err := cli.Create(addr, "")
	if err != nil {
		log.Println(err)
	}

	client.Cmd("reload-timer-config")
}

func TestPublish(t *testing.T) {
	client, err := cli.Create(addr, "")
	if err != nil {
		log.Println(err)
	}
	for i := 0; i < 10000; i++ {
		client.Publish("ax1", strconv.Itoa(i))
	}

	time.Sleep(time.Second)
}
