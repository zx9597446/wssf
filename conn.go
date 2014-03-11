// Copyright 2013 Gary Burd. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wssf

import (
	"errors"
	"time"

	"github.com/gorilla/websocket"
)

type wattingMsg struct {
	mt   int
	data []byte
}

type timerNotify struct {
	fn func()
}

type receiveNotify struct {
	mt   int
	data []byte
	err  error
}

type sendErrNotify struct {
	err error
}

// Connection is an middleman between the websocket Connection and the hub.
type Connection struct {
	// The websocket Connection.
	ws *websocket.Conn

	// Buffered channel of outbound messages.
	send chan wattingMsg

	handler ConnectionHandler

	notifyChan chan interface{}
}

//notify this connection something
//concurrent safe
func (c *Connection) Notify(v interface{}) {
	c.notifyChan <- v
}

//send message to peer, mt: wssf.TextMessage or wssf.BinaryMessage
//concurrent safe
func (c *Connection) Send(mt int, data []byte) {
	m := wattingMsg{mt, data}
	c.send <- m
}

//add a timer function, called from same goroutine with Handler.OnReceived()
//to stop timer, use returned channel
//for example: rc := c.AddTimer(d, fn)
//rc <- true
func (c *Connection) AddTimer(d time.Duration, fn func()) chan bool {
	stop := make(chan bool)
	go func() {
		select {
		case <-time.After(d):
			c.Notify(timerNotify{fn})
		case <-stop: //cancel timer
		}
	}()
	return stop
}

// readPump pumps messages from the websocket Connection to the hub.
func (c *Connection) readPump() {
	defer func() {
		c.handler.OnDisconnected(c)
		h.unregister <- c
		c.ws.Close()
	}()
	c.ws.SetReadLimit(Config.MaxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(Config.PongWait))
	c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(Config.PongWait)); return nil })

	go func() {
		for {
			mt, data, err := c.ws.ReadMessage()
			c.Notify(receiveNotify{mt, data, err})
			if err != nil {
				break
			}
		}
	}()

L:
	for {
		v := <-c.notifyChan
		switch r := v.(type) {
		case timerNotify:
			r.fn()
		case receiveNotify:
			if r.err != nil {
				c.handler.OnError(r.err)
				break L
			}
			if !c.handler.OnReceived(c, r.mt, r.data) {
				break L
			}
		case sendErrNotify:
			if r.err != nil {
				c.handler.OnError(r.err)
				break L
			}
		default:
			c.handler.OnNotify(v)
		}
	}
}

// write writes a message with the given message type and payload.
func (c *Connection) write(mt int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(Config.WriteWait))
	return c.ws.WriteMessage(mt, payload)
}

// writePump pumps messages from the hub to the websocket Connection.
func (c *Connection) writePump() {
	ticker := time.NewTicker(Config.PingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.write(websocket.CloseMessage, []byte{})
				c.Notify(sendErrNotify{errors.New("receive msg from send channel failed")})
				return
			}
			if err := c.write(message.mt, message.data); err != nil {
				c.Notify(sendErrNotify{err})
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				c.Notify(sendErrNotify{err})
				return
			}
		}
	}
}
