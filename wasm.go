package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"syscall/js"
	"time"

	"golang.org/x/crypto/ssh"
	"nhooyr.io/websocket"
)

var console js.Value
var hasData chan bool

func main() {
	hasData = make(chan bool, 1)
	js.Global().Set("hookTerminal", terminalWrapper())
	<-make(chan bool)
}

type TerminalIO struct {
	el        js.Value
	data      []byte
	readIndex int
}

func (t TerminalIO) Write(v []byte) (int, error) {
	t.el.Call("write", string(v))
	return len(v), nil
}

func (t TerminalIO) prompt() {
	t.Write([]byte("\r\n"))
}

func (r *TerminalIO) Read(p []byte) (n int, err error) {
	<-hasData
	copy(p, r.data)
	n = len(r.data)
	r.data = []byte("")
	return
}

func (t *TerminalIO) append(s string) {
	t.data = append(t.data, s...)
	hasData <- true
}

func initWS() net.Conn {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	c, _, err := websocket.Dial(ctx, "ws://localhost:8081/ws", nil)
	if err != nil {
		fmt.Printf("Failed: :%v", err)
	}
	return websocket.NetConn(ctx, c, websocket.MessageBinary)
}

func terminalWrapper() js.Func {
	jsonFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) != 1 {
			return "Invalid arguments"
		}

		writer := &TerminalIO{
			el: args[0],
		}

		var command string

		onData := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			val := args[0].String()
			switch val {
			case "\u0003": // Ctrl+C
				writer.Write([]byte("^C"))
				writer.prompt()
			case "\r": // Enter
				writer.append(command)
				writer.append("\r")
				writer.prompt()
				command = ""
			default:
				command += val
				writer.Write([]byte(val))
			}

			return nil
		})

		args[0].Call("onData", onData)

		go sshWithPassd("localhost:2222", "test", "test", writer)

		return nil
	})

	return jsonFunc
}

func Dial(addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	conn := initWS()
	fmt.Printf("ws: %+v", conn)
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		fmt.Printf("ws: %+v", err)
	}
	return ssh.NewClient(c, chans, reqs), nil
}

func sshWithPassd(addr, user, pass string, writer io.ReadWriter) error {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(pass),
		},
		HostKeyCallback: ssh.HostKeyCallback(func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil }),
	}

	client, err := Dial(addr, config)

	if err != nil {
		return nil
	}
	session, err := client.NewSession()
	if err != nil {
		return nil
	}

	session.Stdout = writer
	session.Stdin = writer
	session.Stderr = writer
	var modes ssh.TerminalModes

	if err := session.RequestPty("xterm", 120, 120, modes); err != nil {
		return nil
	}
	if err := session.Shell(); err != nil {
		return nil
	}
	if err := session.Wait(); err != nil {
		return nil
	}

	return nil
}
