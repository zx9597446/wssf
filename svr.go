// Copyright 2013 Gary Burd. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wssf

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	TextMessage   = websocket.TextMessage
	BinaryMessage = websocket.BinaryMessage
)

const (
	//send buffer length
	defaultSendBufferLen = 256

	// Time allowed to write a message to the peer.
	defaultWriteWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	defaultPongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	defaultPingPeriod = (defaultPongWait * 9) / 10

	// Maximum message size allowed from peer.
	defaultMaxMessageSize = 512
)

type ServerConfig struct {
	// Time allowed to write a message to the peer.
	WriteWait time.Duration

	// Time allowed to read the next pong message from the peer.
	PongWait time.Duration

	// Send pings to peer with this period. Must be less than pongWait.
	PingPeriod time.Duration

	// Maximum message size allowed from peer.
	MaxMessageSize int64

	//send buffer length
	SendBufferLength uint32
}

var Config = ServerConfig{
	defaultWriteWait,
	defaultPongWait,
	defaultPingPeriod,
	defaultMaxMessageSize,
	defaultSendBufferLen,
}

//handler per websocket connection
//all these callback functions called from http.HandlerFunc goroutine
type ConnectionHandler interface {
	//called when connection made
	OnConnected(conn *Connection)
	//called when connection lost
	OnDisconnected(conn *Connection)
	//called when message received
	//return false will close connection
	//mt: wssf.TextMessage or wssf.BinaryMessage
	OnReceived(conn *Connection, mt int, data []byte) bool
	//called when nofication made from another goroutine by Connection.Notify()
	OnNotify(v interface{})
	//called when something goes wrong
	OnError(err error)
}

//broadcast message to all connections
//concuurent safe
func BroadcastMsg(mt int, data []byte) {
	m := wattingMsg{mt, data}
	h.broadcast <- m
}

//serve websocket connection by handler
//like http.Get("/", handler)
func ServeWS(router, method, origin string, handler ConnectionHandler) {
	http.HandleFunc(router, func(w http.ResponseWriter, r *http.Request) {
		serveWsHandler(method, origin, handler, w, r)
	})
}

func serveWsHandler(method, origin string, handler ConnectionHandler, w http.ResponseWriter, r *http.Request) {
	if method != "" && r.Method != method {
		http.Error(w, "Method not allowed", 405)
		return
	}
	if origin != "" && r.Header.Get("Origin") != "http://"+r.Host {
		http.Error(w, "Origin not allowed", 403)
		return
	}
	ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		log.Println(err)
		return
	}
	c := &Connection{send: make(chan wattingMsg, defaultSendBufferLen), ws: ws, handler: handler}
	h.register <- c
	go c.writePump()
	handler.OnConnected(c)
	c.readPump()
}
