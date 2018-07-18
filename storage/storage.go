package storage

import (
	"sync"
	"github.com/gorilla/websocket"
	"sync/atomic"
	"errors"
	"net/http"
)

var storage = make(map[string]*ProxyEntry)
var mutex sync.RWMutex

type ProxyEntry struct {
	path   string
	server *websocket.Conn
	client atomic.Value
	closed atomic.Value
}

func (p *ProxyEntry) IsClosed() (bool) {
	return p.closed.Load().(bool)
}

func (p *ProxyEntry) MarkClose() {
	p.closed.Store(true)
}

func (p *ProxyEntry) CloseClient() {
	if p.client.Load() != nil {
		p.client.Load().(*websocket.Conn).Close()
	}
}

func (p *ProxyEntry) CloseAll() {
	p.MarkClose()
	p.CloseClient()
	p.server.Close()
}

func (p *ProxyEntry) SendClient(messageType int, data []byte) (error) {
	if p.client.Load() != nil {
		return p.client.Load().(*websocket.Conn).WriteMessage(messageType, data)
	}
	return errors.New("client is nil")
}

func (p *ProxyEntry) SendServer(messageType int, data []byte) (error) {
	if p.client.Load() != nil {
		return p.server.WriteMessage(messageType, data)
	}
	return errors.New("client is nil")
}

func (p *ProxyEntry) NewClient(conn *websocket.Conn) {
	p.client.Store(conn)
}

func InitNewEntry(serverConnection *websocket.Conn, r *http.Request) (*ProxyEntry) {
	mutex.Lock()
	oldProxyEntry, oldEntryExists := storage[r.URL.Path]
	newEntry := &ProxyEntry{
		r.URL.Path,
		serverConnection,
		atomic.Value{},
		atomic.Value{},
	}
	newEntry.closed.Store(false)
	storage[r.URL.Path] = newEntry
	mutex.Unlock()

	if oldEntryExists {
		oldProxyEntry.CloseAll()
	}
	return newEntry
}
