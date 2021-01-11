package main

import (
	"log"
	"testing"
	"time"
	"zhub/cli"
)

func TestCli(t *testing.T) {
	//client, err := cli.Create("39.108.56.246:7070", "")
	client, err := cli.Create("47.111.150.118:6066", "")
	//client, err := cli.Create("127.0.0.1:1216", "topic-x")
	if err != nil {
		log.Fatal(err)
	}

	// 订阅主题 消息
	client.Subscribe("a", func(v string) {
		log.Println("收到主题 a 消息 " + v)
	})

	// 定时调度
	client.Timer("a", "*/5 * * * * *", func() {
		log.Println("收到 t------------------x 定时消息")
	})

	/*go func() {
		for i := 0; i < 100000; i++ {
			client.Publish("a", strconv.Itoa(i))
			time.Sleep(time.Second)
		}
	}()*/

	client.Subscribe("a", func(v string) {
		log.Println("收到主题 a 消息 " + v)
	})
	client.Daly("a", "x", 3000)

	time.Sleep(time.Hour * 3)
}

func TestTimer(t *testing.T) {
	go func() {
		client, _ := cli.Create("127.0.0.1:1216", "topic-x")
		client.Timer("t", "*/3 * * * * *", func() {
			log.Println("=======收到 t 定时消息")
		})

		client.Timer("t------------------x", "*/3 * * * * *", func() {
			log.Println("收到 t------------------x 定时消息")
		})
	}()

	time.Sleep(time.Second * 5)
	go func() {
		client, _ := cli.Create("127.0.0.1:1216", "topic-x")
		client.Timer("t", "*/5 * * * * *", func() {
			log.Println("-------收到 t 定时消息")
		})
	}()

	time.Sleep(time.Hour * 3)
}
