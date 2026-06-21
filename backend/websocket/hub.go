package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Hub struct{}

func NewHub() *Hub {
	return &Hub{}
}

func (h *Hub) Run() {}

func (h *Hub) HandleTraffic(c *gin.Context) {
	h.proxyMihomoWebSocket(c, "/traffic")
}

func (h *Hub) HandleLogs(c *gin.Context) {
	h.proxyMihomoWebSocket(c, "/logs")
}

func (h *Hub) HandleConnections(c *gin.Context) {
	h.proxyMihomoWebSocket(c, "/connections")
}

type connectionsPayload struct {
	DownloadTotal   int64                    `json:"downloadTotal,omitempty"`
	UploadTotal     int64                    `json:"uploadTotal,omitempty"`
	Connections     []map[string]interface{} `json:"connections,omitempty"`
	ConnectionCount int                      `json:"connectionCount"`
	Truncated       bool                     `json:"truncated,omitempty"`
	Limit           int                      `json:"limit,omitempty"`
}

func (h *Hub) proxyMihomoWebSocket(c *gin.Context, path string) {
	log.Printf("[WebSocket] proxy request: %s", path)

	clientConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WebSocket] upgrade failed: %v", err)
		return
	}
	defer clientConn.Close()

	mihomoURL := "ws://127.0.0.1:9090" + path
	queryString := c.Request.URL.RawQuery
	if queryString != "" {
		mihomoURL += "?" + queryString
	}

	log.Printf("[WebSocket] dialing Mihomo: %s", mihomoURL)
	mihomoConn, _, err := websocket.DefaultDialer.Dial(mihomoURL, nil)
	if err != nil {
		log.Printf("[WebSocket] Mihomo dial failed: %v", err)
		_ = clientConn.WriteMessage(websocket.TextMessage, []byte(`{"error":"cannot connect to Mihomo"}`))
		return
	}
	defer mihomoConn.Close()

	done := make(chan struct{})
	isConnections := path == "/connections"
	connectionLimit := clampInt(queryInt(c, "limit", 300), 50, 1000)
	connectionSummary := queryBool(c, "summary")
	connectionInterval := time.Duration(clampInt(queryInt(c, "interval", 1500), 500, 5000)) * time.Millisecond
	lastConnectionSend := time.Time{}

	go func() {
		defer close(done)
		for {
			msgType, msg, err := mihomoConn.ReadMessage()
			if err != nil {
				return
			}
			if isConnections {
				now := time.Now()
				if !lastConnectionSend.IsZero() && now.Sub(lastConnectionSend) < connectionInterval {
					continue
				}
				lastConnectionSend = now
				if filtered, ok := compactConnectionsMessage(msg, connectionLimit, connectionSummary); ok {
					msg = filtered
				}
			}
			_ = clientConn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := clientConn.WriteMessage(msgType, msg); err != nil {
				return
			}
		}
	}()

	go func() {
		for {
			msgType, msg, err := clientConn.ReadMessage()
			if err != nil {
				return
			}
			if err := mihomoConn.WriteMessage(msgType, msg); err != nil {
				return
			}
		}
	}()

	<-done
}

func queryInt(c *gin.Context, name string, fallback int) int {
	value := c.Query(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func queryBool(c *gin.Context, name string) bool {
	switch c.Query(name) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func compactConnectionsMessage(msg []byte, limit int, summary bool) ([]byte, bool) {
	var payload connectionsPayload
	if err := json.Unmarshal(msg, &payload); err != nil {
		return nil, false
	}
	total := len(payload.Connections)
	payload.ConnectionCount = total
	if summary {
		payload.Connections = nil
	} else if total > limit {
		sort.SliceStable(payload.Connections, func(i, j int) bool {
			return connectionBytes(payload.Connections[i]) > connectionBytes(payload.Connections[j])
		})
		payload.Connections = payload.Connections[:limit]
		payload.Truncated = true
		payload.Limit = limit
	}
	filtered, err := json.Marshal(payload)
	if err != nil {
		return nil, false
	}
	return filtered, true
}

func connectionBytes(item map[string]interface{}) float64 {
	return numericField(item, "upload") + numericField(item, "download")
}

func numericField(item map[string]interface{}, key string) float64 {
	switch value := item[key].(type) {
	case float64:
		return value
	case int:
		return float64(value)
	case int64:
		return float64(value)
	case json.Number:
		number, _ := value.Float64()
		return number
	default:
		return 0
	}
}
