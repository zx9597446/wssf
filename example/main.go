package main

import (
	"log"
	"net/http"
	"time"

	"github.com/zx9597446/wssf"
)

type MyConnectionHandler struct {
}

func (h *MyConnectionHandler) OnConnected(conn *wssf.Connection) {
	conn.AddTimer(60*time.Second, h.onTimer)
}

func (h *MyConnectionHandler) onTimer() {
	log.Println("onTimer")
}

func (h *MyConnectionHandler) OnDisconnected(conn *wssf.Connection) {
}

func (h *MyConnectionHandler) OnReceived(conn *wssf.Connection, mt int, data []byte) bool {
	wssf.BroadcastMsg(mt, data)
	return true
}

func (h *MyConnectionHandler) OnError(err error) {
	log.Println(err)
}

func (h *MyConnectionHandler) OnNotify(v interface{}) {
}

func NewHandler() *MyConnectionHandler {
	return &MyConnectionHandler{}
}

func main() {
	wssf.ServeWS("/ws", "GET", "", NewHandler())
	log.Fatal(http.ListenAndServe(":8080", nil))
}
