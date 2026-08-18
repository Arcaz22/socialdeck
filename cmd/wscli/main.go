package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

func main() {
	url := flag.String("url", "", "websocket URL")
	flag.Parse()

	if *url == "" {
		log.Fatal("missing -url")
	}

	conn, _, err := websocket.DefaultDialer.Dial(*url, nil)
	if err != nil {
		log.Fatalf("connect websocket: %v", err)
	}
	defer conn.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				fmt.Fprintf(os.Stderr, "read websocket: %v\n", err)
				return
			}
			fmt.Println(string(message))
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if err := conn.WriteMessage(websocket.TextMessage, []byte(line)); err != nil {
			log.Fatalf("write websocket: %v", err)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("read stdin: %v", err)
	}

	_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	<-done
}
