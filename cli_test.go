package main

import (
	"log"
	"strconv"
	"testing"
	"time"
	"zhub/cli"
)

var (
	addr = "47.111.150.118:6066"
	//addr = "127.0.0.1:1216"
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

	go func() {
		for i := 0; i < 100000; i++ {
			client.Publish("ax", strconv.Itoa(i))
			time.Sleep(time.Second)
		}
	}()

	client.Subscribe("a", func(v string) {
		log.Println("收到主题 a 消息 " + v)
	})
	client.Daly("a", "x", 3000)

	time.Sleep(time.Hour * 3)
}

func TestTimer(t *testing.T) {
	go func() {
		client, _ := cli.Create(addr, "topic-x")
		client.Timer("a", func() {
			log.Println("client-1 收到 a 的定时消息")
		})
	}()

	time.Sleep(time.Second * 5)
	go func() {
		client, _ := cli.Create(addr, "topic-x")
		client.Timer("a", func() {
			log.Println("client-2 收到 a 的定时消息")
		})
	}()

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

	client.Publish("ax", "a")

	time.Sleep(time.Second)
}
