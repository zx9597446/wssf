package svrframework

import (
	"log"
	"runtime/debug"
	"time"
)

type HandleMsgFunc func([]byte)
type HandleEventFunc func(first, second interface{})
type TimerFunc func()

type msg struct {
	conn *Connection
	data []byte
	cmd  uint32
}

type event struct {
	cmd           uint64
	first, second interface{}
}

type Dispatcher struct {
	msg_map           map[uint32]HandleMsgFunc
	event_map         map[uint64]HandleEventFunc
	event_chan        chan event
	disconnected_chan chan *Connection
	disconnected      bool
	msg_chan          chan msg
	handler           ConnectionHandler
	timer_chan        chan TimerFunc
}

func (h *Dispatcher) init(hdl ConnectionHandler) {
	h.msg_map = map[uint32]HandleMsgFunc{}
	h.event_map = map[uint64]HandleEventFunc{}
	h.handler = hdl
	h.msg_chan = make(chan msg, 100)
	h.event_chan = make(chan event, 100)
	h.disconnected_chan = make(chan *Connection)
	h.disconnected = false
	h.timer_chan = make(chan TimerFunc, 100)
}

func (h *Dispatcher) AddMsgFunc(cmd uint32, f HandleMsgFunc) {
	h.msg_map[cmd] = f
}
func (h *Dispatcher) AddEventFunc(event uint64, f HandleEventFunc) {
	h.event_map[event] = f
}
func (h *Dispatcher) AddTimerFunc(ms uint64, fn TimerFunc) chan bool {
	stop := make(chan bool)
	go func() {
		select {
		case <-time.After(time.Duration(ms) * time.Millisecond):
			h.timer_chan <- fn
		case <-stop: //cancel timer
		}
	}()
	return stop
}

func (h *Dispatcher) loop() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("loop failed:", err)
			debug.PrintStack()
		}
	}()

	for {
		select {
		case m := <-h.msg_chan:
			h.handleMsg(m.cmd, m.data, m.conn)
		case e := <-h.event_chan:
			h.handleEvent(e)
		case fn := <-h.timer_chan:
			fn()
		case c := <-h.disconnected_chan:
			if !h.disconnected {
				h.handler.OnDisconnected(c)
				h.disconnected = true
				return
			}
		}
	}
}

func (h *Dispatcher) handleMsg(cmd uint32, data []byte, conn *Connection) {
	if f, ok := h.msg_map[cmd]; ok {
		f(data)
	}
}

func (h *Dispatcher) handleEvent(e event) {
	if f, ok := h.event_map[e.cmd]; ok {
		f(e.first, e.second)
	}
}

func (h *Dispatcher) NotifyEvent(cmd uint64, first, second interface{}) {
	e := event{cmd, first, second}
	h.event_chan <- e
}
