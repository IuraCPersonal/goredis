package client

import (
	"context"
	"fmt"
	"log"
	"testing"
)

func TestNewClient(t *testing.T) {
	client, err := New("localhost:5001")

	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < 10; i++ {

		if err := client.Set(context.Background(), fmt.Sprintf("foo_%d", i), "bar"); err != nil {
			log.Fatal(err)
		}

		val, err := client.Get(context.Background(), fmt.Sprintf("foo_%d", i))

		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("got this back =>", val)
	}
}
