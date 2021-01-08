package main

import (
	"log"
	"strconv"
	"testing"
	"time"
	"zhub/cli"
)

func TestName(t *testing.T) {
	//client, err := cli.Create("39.108.56.246:1216", "")
	client, err := cli.Create("127.0.0.1:1216", "")
	if err != nil {
		log.Fatal(err)
	}

	client.Subscribe("a-1", func(v string) {
		log.Println(v)
	})

	client.Timer("t", "* * * * * *")

	go func() {
		for i := 0; i < 50000; i++ {
			client.Publish("a-1", strconv.Itoa(i))
			time.Sleep(time.Second)
		}
	}()

	time.Sleep(time.Hour * 3)
}
