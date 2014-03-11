// Copyright 2013 Gary Burd. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wssf

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

const (
	MsgTypeText   = websocket.TextMessage
	MsgTypeBinary = websocket.BinaryMessage
)

const (
	sendBufferLen = 256
)

//handler per websocket connection
//all these callback functions called from http.HandlerFunc goroutine
type ConnectionHandler interface {
	OnConnected(conn *Connection)
	OnDisconnected(conn *Connection)
	//return false will close connection
	OnReceived(conn *Connection, mt int, data []byte) bool
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
	c := &Connection{send: make(chan wattingMsg, sendBufferLen), ws: ws, handler: handler}
	h.register <- c
	go c.writePump()
	handler.OnConnected(c)
	c.readPump()
}
