package main

import (
	"context"
	"fmt"
	"goredis/client"
	"log"
	"sync"
	"testing"
	"time"
)

func TestServerWithClients(t *testing.T) {
	server := NewServer(Config{})
	go func() {
		log.Fatal(server.Start())
	}()

	time.Sleep(time.Second)

	nClients := 10
	wg := sync.WaitGroup{}
	wg.Add(nClients)

	for i := 0; i < nClients; i++ {
		go func(it int) {
			client, err := client.New("localhost:5001")

			if err != nil {
				log.Fatal(err)
			}

			defer client.Close()

			key := fmt.Sprintf("client_%d_foo", it)
			value := fmt.Sprintf("client_%d_bar", it)

			if err := client.Set(context.Background(), key, value); err != nil {
				log.Fatal(err)
			}

			val, err := client.Get(context.Background(), key)

			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("client %d got this back => %s\n", it, val)
			wg.Done()
		}(i)
	}

	wg.Wait()

	time.Sleep(time.Second)

	if len(server.peers) != 0 {
		t.Fatalf("expected 0 peers but got %d", len(server.peers))
	}

}
