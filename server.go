package main

import (
	"context"
	// "fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"nhooyr.io/websocket"
)

func main() {
	http.HandleFunc("/ws", wsHandler)
	http.Handle("/", http.FileServer(http.Dir("./")))
	log.Fatal(http.ListenAndServe(":8081", nil))

}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}

	defer c.Close(websocket.StatusInternalError, "internal error")

	ctx, cancel := context.WithTimeout(r.Context(), time.Minute)
	defer cancel()

	conn := websocket.NetConn(ctx, c, websocket.MessageBinary)
	handleRequest(conn, "localhost:2222")

	for {
	}

	c.Close(websocket.StatusNormalClosure, "")
	return
}

func handleRequest(conn net.Conn, proxyTo string) {
	log.Printf("handling %s", proxyTo)
	proxy, err := net.Dial("tcp", proxyTo)
	if err != nil {
		panic(err)
	}

	go copyIO(proxy, conn)
	go copyIO(conn, proxy)

}

func copyIO(src, dest net.Conn) {
	defer src.Close()
	defer dest.Close()
	io.Copy(src, dest)
}
