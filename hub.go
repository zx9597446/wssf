package svrframework

// hub maintains the set of active Connections and broadcasts messages to the
// Connections.
type hub struct {
	// Registered Connections.
	Connections map[*Connection]bool

	// Inbound messages from the Connections.
	broadcast chan []byte

	// Register requests from the Connections.
	register chan *Connection

	// Unregister requests from Connections.
	unregister chan *Connection
}

var gHub = hub{
	broadcast:   make(chan []byte),
	register:    make(chan *Connection),
	unregister:  make(chan *Connection),
	Connections: make(map[*Connection]bool),
}

func (h *hub) run() {
	for {
		select {
		case c := <-h.register:
			h.Connections[c] = true
		case c := <-h.unregister:
			delete(h.Connections, c)
			close(c.send)
		case m := <-h.broadcast:
			for c := range h.Connections {
				select {
				case c.send <- m:
				default:
					close(c.send)
					delete(h.Connections, c)
				}
			}
		}
	}
}
