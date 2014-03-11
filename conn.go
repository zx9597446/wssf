// Copyright 2013 Gary Burd. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wssf

import (
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

type wattingMsg struct {
	mt   int
	data []byte
}

// Connection is an middleman between the websocket Connection and the hub.
type Connection struct {
	// The websocket Connection.
	ws *websocket.Conn

	// Buffered channel of outbound messages.
	send chan wattingMsg

	handler ConnectionHandler

	timer chan func()
}

//send message to peer
//concurrent safe
func (c *Connection) Send(mt int, data []byte) {
	m := wattingMsg{mt, data}
	c.send <- m
}

//sometime need a timer function called from same goroutine with OnReceived
//to stop timer, use returned channel
//for example: rc := c.AddTimer(d, fn)
//rc <- true
func (c *Connection) AddTimer(d time.Duration, fn func()) chan bool {
	stop := make(chan bool)
	go func() {
		select {
		case <-time.After(d):
			c.timer <- fn
		case <-stop: //cancel timer
		}
	}()
	return stop
}

// readPump pumps messages from the websocket Connection to the hub.
func (c *Connection) readPump() {
	defer func() {
		//notify disconnected
		c.handler.OnDisconnected(c)
		h.unregister <- c
		c.ws.Close()
	}()
	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		select {
		case fn := <-c.timer:
			fn()
		}
		mt, data, err := c.ws.ReadMessage()
		if err != nil {
			break
		}
		if !c.handler.OnReceived(c, mt, data) {
			break
		}
	}
}

// write writes a message with the given message type and payload.
func (c *Connection) write(mt int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}

// writePump pumps messages from the hub to the websocket Connection.
func (c *Connection) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.write(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.write(message.mt, message.data); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}
