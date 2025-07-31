// internal/httpserver/server.go
package httpserver

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

var tmpl = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html data-theme="dark">
<head>
	<meta charset="utf-8">
	<link href="https://cdn.jsdelivr.net/npm/daisyui@4.4.20/dist/full.css" rel="stylesheet" type="text/css" />
	<script src="https://cdn.tailwindcss.com"></script>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>Mic Volume Controller</title>
</head>
<body class="p-4">
	<div class="max-w-md mx-auto space-y-6">
		<h1 class="text-2xl font-bold">Mic Control</h1>
		<p class="text-sm text-gray-400 break-words">Device: {{.Device}}</p>
		<div class="form-control">
			<label class="label">
				<span class="label-text">Interval (ms)</span>
			</label>
			<div class="flex flex-col sm:flex-row items-center gap-2">
				<input id="interval" type="range" min="100" max="10000" step="100" class="range range-primary w-full">
				<output id="intervalOut" class="text-right w-16"></output>
			</div>
		</div>
		<div class="form-control">
			<label class="label">
				<span class="label-text">Volume (%)</span>
			</label>
			<div class="flex flex-col sm:flex-row items-center gap-2">
				<input id="volume" type="range" min="0" max="100" class="range range-secondary w-full">
				<output id="volumeOut" class="text-right w-16"></output>
			</div>
		</div>
	</div>
	<script>
		const ws = new WebSocket("ws://" + location.host + "/ws")
		const interval = document.getElementById("interval")
		const intervalOut = document.getElementById("intervalOut")
		const volume = document.getElementById("volume")
		const volumeOut = document.getElementById("volumeOut")

		ws.onopen = () => {
			interval.oninput = () => {
				intervalOut.value = interval.value
					ws.send("interval:" + interval.value)
			}
			volume.oninput = () => {
				volumeOut.value = volume.value
					ws.send("volume:" + volume.value)
			}
		}

		ws.onmessage = (msg) => {
			const data = JSON.parse(msg.data)
			interval.value = data.interval
			intervalOut.value = data.interval
			volume.value = data.volume
			volumeOut.value = data.volume
		}
	</script>
</body>
</html>`))

var deviceName atomic.Value
var upgrader = websocket.Upgrader{}
var conns sync.Map // key: *websocket.Conn, value: struct{}

func SetDeviceName(name string) {
	deviceName.Store(name)
}

func broadcastUpdate(interval, volume int64) {
	msg := struct {
		Interval int64 `json:"interval"`
		Volume   int64 `json:"volume"`
	}{interval, volume}
	conns.Range(func(key, _ any) bool {
		conn := key.(*websocket.Conn)
		err := conn.WriteJSON(msg)
		if err != nil {
			conns.Delete(conn)
			conn.Close()
		}
		return true
	})
}

func Start(addr string, interval, volume *atomic.Int64) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		name := ""
		if dn := deviceName.Load(); dn != nil {
			name = dn.(string)
		}
		tmpl.Execute(w, struct{ Device string }{name})
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("WebSocket upgrade error:", err)
			return
		}
		conns.Store(conn, struct{}{})
		defer func() {
			conns.Delete(conn)
			conn.Close()
		}()

		conn.WriteJSON(struct {
			Interval int64 `json:"interval"`
			Volume   int64 `json:"volume"`
		}{interval.Load(), volume.Load()})

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("WebSocket read error:", err)
				return
			}
			parts := strings.SplitN(string(msg), ":", 2)
			if len(parts) != 2 {
				continue
			}
			updated := false
			switch parts[0] {
			case "interval":
				if ms, err := strconv.ParseInt(parts[1], 10, 64); err == nil && ms >= 100 && ms <= 10000 {
					interval.Store(ms)
					updated = true
				}
			case "volume":
				if p, err := strconv.ParseInt(parts[1], 10, 64); err == nil && p >= 0 && p <= 100 {
					volume.Store(p)
					updated = true
				}
			}
			if updated {
				broadcastUpdate(interval.Load(), volume.Load())
			}
		}
	})

	log.Println("Serving HTML control UI at", addr)
	http.ListenAndServe(addr, nil)
}
