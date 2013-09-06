package svrframework

import (
	"bytes"
	"encoding/binary"
	"github.com/garyburd/go-websocket/websocket"
	"io/ioutil"
	"log"
	"time"
)

const (
	// Time allowed to write a message to the client.
	writeWait = 10 * time.Second

	// Time allowed to read the next message from the client.
	readWait = 60 * time.Second

	// Send pings to client with this period. Must be less than readWait.
	pingPeriod = (readWait * 9) / 10

	// Maximum message size allowed from client.
	maxMessageSize = 512
)

// Connection is an middleman between the websocket Connection and the hub.
type Connection struct {
	Dispatcher
	// The websocket Connection.
	ws *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

// readPump pumps messages from the websocket Connection to the hub.
func (c *Connection) readPump() {
	defer func() {
		gHub.unregister <- c
		c.ws.Close()
		c.disconnected_chan <- c
	}()
	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(readWait))
	for {
		op, r, err := c.ws.NextReader()
		if err != nil {
			break
		}
		switch op {
		case websocket.OpPong:
			c.ws.SetReadDeadline(time.Now().Add(readWait))
		case websocket.OpText:
			/*message, err := ioutil.ReadAll(r)*/
			/*if err != nil {*/
			/*break*/
			/*}*/
			/*log.Println(message, err)*/
			/*h.broadcast <- message*/
		case websocket.OpBinary:
			message, err := ioutil.ReadAll(r)
			if err != nil {
				break
			}
			var cmd uint32
			/*err = msgpack.Unmarshal(message[0:5], &cmd, nil)*/
			err = binary.Read(bytes.NewBuffer(message[0:4]), binary.BigEndian, &cmd)
			if err != nil {
				log.Println(err)
				break
			}
			c.msg_chan <- msg{c, message[4:], cmd}
		}
	}
}

// write writes a message with the given opCode and payload.
func (c *Connection) write(opCode int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(opCode, payload)
}

// writePump pumps messages from the hub to the websocket Connection.
func (c *Connection) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
		c.disconnected_chan <- c
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.write(websocket.OpClose, []byte{})
				return
			}
			if err := c.write(websocket.OpBinary, message); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.OpPing, []byte{}); err != nil {
				return
			}
		}
	}
}

//send message(thread safe)
func (c *Connection) SendMsg(cmd uint32, body []byte) {
	header := new(bytes.Buffer)
	binary.Write(header, binary.BigEndian, cmd)
	if body != nil {
		all := make([]byte, header.Len()+len(body))
		copy(all[0:header.Len()], header.Bytes())
		copy(all[header.Len():], body)

		c.send <- all
	} else {
		c.send <- header.Bytes()
	}
}

func (c *Connection) GetHandler() ConnectionHandler {
	return c.handler
}
