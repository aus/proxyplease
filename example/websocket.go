package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	proxyplease "github.com/aus/proxy-please"
	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "echo.websocket.org", "http service address")

func main() {
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/"}
	log.Printf("connecting to %s", u.String())

	// Examples

	// SOCKS with authentication example
	//proxyURL, _ := url.Parse("socks5://localhost:8888")
	//dialContext := proxyplease.NewProxyDialContext(proxyplease.Proxy{Url: proxyURL, Username: "foo", Password: "bar"})

	// Assume proxy from environment. Try to authenticate with these credentials
	//dialContext := proxyplease.NewProxyDialContext(proxyplease.Proxy{Username: "foo", Password: "bar"})

	// Specify with a domain to enable NTLM authentication
	dialContext := proxyplease.NewDialContext(proxyplease.Proxy{Username: "foo", Password: "bar", Domain: "EXAMPLE"})

	d := websocket.Dialer{
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		HandshakeTimeout: 45 * time.Second,
		NetDialContext:   dialContext,
	}
	c, _, err := d.Dial(u.String(), nil)

	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", message)
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				log.Println("write:", err)
				return
			}
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}
