package main

import (
	"html/template"
	"log"
	"net/http"

	"github.com/gomarkdown/markdown"
	_ "github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/gorilla/websocket"
)

var allTemplates = template.Must(
	template.Must(
		template.ParseGlob("templates/*.html"),
	).ParseGlob("templates/_partials/*"))

func serveChat(res http.ResponseWriter, req *http.Request) {
	allTemplates.ExecuteTemplate(res, "chat.html", nil)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func mdToHTML(md []byte) []byte {
	// create markdown parser with extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	// create HTML renderer with extensions
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	return markdown.Render(doc, renderer)
}

var clients = make(map[*websocket.Conn]bool)
var OAresponse = make(chan Message)

var roleMap = make(map[string]*websocket.Conn)

var userHistory = make(map[*websocket.Conn][]map[string]string)

type Message struct {
	Text string `json:"text"`
	User string `json:"user"`
	Role string `json:"role"`
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	clients[ws] = true
	for {
		var msg Message
		err = ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, ws)
			break
		}
		roleMap[msg.User] = ws
		OAresponse <- msg
		userHistory[ws] = append(userHistory[ws], map[string]string{
			"role":    "user",
			"content": msg.Text,
		})
		llmMessage := GetOpenAIMessageStr(userHistory[ws])
		openaimsg := string(mdToHTML([]byte(llmMessage)))
		userHistory[ws] = append(userHistory[ws], map[string]string{
			"role":    "system",
			"content": llmMessage,
		})
		OAresponse <- Message{Text: openaimsg, User: msg.User, Role: LLM}
	}
}

func handleMessages() {
	for {
		msg := <-OAresponse
		conn := roleMap[msg.User]
		err := conn.WriteJSON(msg)
		if err != nil {
			log.Printf("error: %v", err)
			conn.Close()
			delete(clients, conn)
		}
	}
}

func main() {
	http.HandleFunc("/ws", handleConnections)
	http.HandleFunc("/chat", serveChat)

	go handleMessages()

	log.Println("http server started on :8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
