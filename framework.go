package svrframework

import (
	"flag"
	"github.com/garyburd/go-websocket/websocket"
	"log"
	"net/http"
)

type ConnectionHandler interface {
	OnConnected(*Connection)
	OnDisconnected(*Connection)
}

type NewConnectionHandlerFunc func() ConnectionHandler

var addr = flag.String("addr", ":80", "http service address")
var gNewFunc NewConnectionHandlerFunc

func Run(url string, factory NewConnectionHandlerFunc) {
	if url == "" {
		url = "/"
	}
	log.Printf("serve websocket on %s\n", url)
	gNewFunc = factory
	flag.Parse()
	go gHub.run()
	http.HandleFunc(url, serveWs)
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// serverWs handles webocket requests from the client.
func serveWs(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		log.Println("Method not allowed")
		http.Error(w, "Method not allowed", 405)
		return
	}
	/*if r.Header.Get("Origin") != "http://"+r.Host {*/
	/*log.Println("Origin not allowed")*/
	/*http.Error(w, "Origin not allowed", 403)*/
	/*return*/
	/*}*/
	ws, err := websocket.Upgrade(w, r.Header, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		log.Println("Not a websocket handshake")
		http.Error(w, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		log.Println(err)
		return
	}
	h := gNewFunc()
	c := &Connection{send: make(chan []byte, 256), ws: ws}
	c.init(h)
	c.handler.OnConnected(c)
	gHub.register <- c
	go c.writePump()
	go c.loop()
	c.readPump()
}
