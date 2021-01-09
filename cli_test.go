package main

import (
	"log"
	"strconv"
	"testing"
	"time"
	"zhub/cli"
)

func TestCli(t *testing.T) {
	//client, err := cli.Create("39.108.56.246:1216", "")
	client, err := cli.Create("127.0.0.1:1216", "")

	if err != nil {
		log.Fatal(err)
	}

	// 订阅主题 消息
	client.Subscribe("a", func(v string) {
		log.Println("收到主题 a 消息 " + v)
	})

	//
	client.Timer("t", "* * * * * *")

	go func() {
		for i := 0; i < 50000; i++ {
			client.Publish("a", strconv.Itoa(i))
			time.Sleep(time.Second)
		}
	}()

	time.Sleep(time.Hour * 3)
}
