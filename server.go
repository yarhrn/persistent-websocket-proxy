package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"github.com/gorilla/websocket"
	"net/url"
	s "github.com/yarhrn/persistent-websocket-proxy/storage"
)

var addr = flag.String("addr", "localhost:8080", "http service address")
var upgrader = websocket.Upgrader{} // use default options

func proxy(w http.ResponseWriter, r *http.Request) {

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return

	}
	entry := newConnection(r, c)

	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read1:", err)
			entry.CloseAll()
			break
		}

		log.Printf("recv1: %s", message)
		if entry.SendClient(mt, message) != nil {
			log.Println("write1:", err)
			entry.CloseClient()
			go connectToBackend(entry)
		}

	}
}

func newConnection(r *http.Request, conn *websocket.Conn) (*s.ProxyEntry) {
	entry := s.InitNewEntry(conn, r)
	go connectToBackend(entry)
	return entry
}

func connectToBackend(entry *s.ProxyEntry) {
	if entry.IsClosed() {
		return
	}
	u := url.URL{Scheme: "ws", Host: "localhost:8099", Path: "/proxy"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Println("dial:", err)
		if !entry.IsClosed() {
			go connectToBackend(entry)
		}
		return
	}
	entry.NewClient(c)

	go func() {
		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read2:", err)
				c.Close()
				if !entry.IsClosed() {
					go connectToBackend(entry)
				}
				break
			}

			log.Printf("recv2: %s", message)

			err = entry.SendServer(mt, message)
			if err != nil {
				log.Println("write2:", err)
				c.Close()
			}

		}
	}()
}

func home(w http.ResponseWriter, r *http.Request) {
	homeTemplate.Execute(w, "ws://"+r.Host+"/proxy")
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/adminka", home)
	http.HandleFunc("/", proxy)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

var homeTemplate = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<script>  
window.addEventListener("load", function(evt) {
    var output = document.getElementById("output");
    var input = document.getElementById("input");
    var ws;
    var print = function(message) {
        var d = document.createElement("div");
        d.innerHTML = message;
        output.appendChild(d);
    };
    document.getElementById("open").onclick = function(evt) {
        if (ws) {
            return false;
        }
        ws = new WebSocket("{{.}}");
        ws.onopen = function(evt) {
            print("OPEN");
        }
        ws.onclose = function(evt) {
            print("CLOSE");
            ws = null;
        }
        ws.onmessage = function(evt) {
            print("RESPONSE: " + evt.data);
        }
        ws.onerror = function(evt) {
            print("ERROR: " + evt.data);
        }
        return false;
    };
    document.getElementById("send").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        print("SEND: " + input.value);
        ws.send(input.value);
        return false;
    };
    document.getElementById("markClose").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        ws.markClose();
        return false;
    };
});
</script>
</head>
<body>
<table>
<tr><td valign="top" width="50%">
<p>Click "Open" to create a connection to the server, 
"Send" to send a message to the server and "Close" to markClose the connection. 
You can change the message and send multiple times.
<p>
<form>
<button id="open">Open</button>
<button id="markClose">Close</button>
<p><input id="input" type="text" value="Hello world!">
<button id="send">Send</button>
</form>
</td><td valign="top" width="50%">
<div id="output"></div>
</td></tr></table>
</body>
</html>
`))
