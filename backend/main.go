package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// '{ model: "schroneko/gemma-2-2b-jpn-it:latest", "promp" : "what is study in japanese", "stream": false }'

const (
	OLLAMA_URL    = "http://localhost:11434/api/generate"
	OLLAMA_MODEL  = "schroneko/gemma-2-2b-jpn-it:latest"
	OLLAMA_STREAM = false
)

func constructPrompt(word string) string {
	return fmt.Sprintf(`You are an AI designed to generate flashcards for learning Japanese. 

**Instructions:**
Generate a flashcard in **structured JSON format only** based on the given Japanese word. The flashcard should include:
- The **word** in Kanji (if available).
- The **pronunciation** in Hiragana/Katakana and no english characters.
- The **meaning** in English.
- An **example sentence** in Japanese.
- The **English translation** of the example sentence.

Here is an example of the expected output:

{
  "front": {
    "word": "勉強",
    "pronunciation": "べんきょう"
  },
  "back": {
    "meaning": "Study, Learning",
    "example_sentence": {
      "japanese": "毎日、日本語を勉強しています。",
      "english": "I study Japanese every day."
    }
  }
}

Now, generate a flashcard for the following word:
**Word:** %s`, word)
}

type OllamaRequest struct {
	client  *http.Client
	request *http.Request
}

func constructFlashCardRequest(prompt string) (*OllamaRequest, error) {
	requestBody, err := json.Marshal(map[string]interface{}{
		"model":  OLLAMA_MODEL,
		"prompt": prompt,
		"stream": OLLAMA_STREAM,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", OLLAMA_URL, bytes.NewBuffer(requestBody))
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}

	return &OllamaRequest{
		client:  client,
		request: req,
	}, err
}

type Router struct {
	mux    *http.ServeMux
	prefix string
}

type Server struct {
	Addr string
}

func NewServer(addr string) *Server {
	return &Server{
		Addr: addr,
	}
}

type CreateRequest struct {
	Word string `json:"word"`
}

type OllamaResponse struct {
	Response string `json:"response"`
}

func createFlashCardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not accepted", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Flashcard could not be generated", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var request CreateRequest
	err = json.Unmarshal(body, &request)
	if err != nil {
		http.Error(w, "Flashcard could not be generated", http.StatusInternalServerError)
		return
	}

	prompt := constructPrompt(request.Word)
	ollamaRequest, err := constructFlashCardRequest(prompt)
	if err != nil {
		http.Error(w, "Flashcard could not be generated", http.StatusInternalServerError)
	}

	res, err := ollamaRequest.client.Do(ollamaRequest.request)
	if err != nil {
		http.Error(w, "Flashcard could not be generated", http.StatusInternalServerError)
	}

	ollamaBody, err := io.ReadAll(res.Body)

	var response OllamaResponse
	err = json.Unmarshal(ollamaBody, &response)
	if err != nil {
		http.Error(w, "Flashcard could not be generated", http.StatusInternalServerError)
	}

	w.Write([]byte(response.Response))
}

func NewFlashCardRouter(prefix string) *Router {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("Flashcard root\n")) })
	mux.HandleFunc("/create", createFlashCardHandler)
	return &Router{
		mux:    mux,
		prefix: prefix,
	}
}

func (s *Server) Run() error {
	mux := http.NewServeMux()

	// catch all
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("Home\n")) })

	// flash card router
	flashCardRouter := NewFlashCardRouter("/api/flash_card")
	mux.Handle(flashCardRouter.prefix+"/", http.StripPrefix(flashCardRouter.prefix, flashCardRouter.mux))

	middleware := ChainMiddleware(
		LoggerMiddleware,
		// AuthMiddleware,
	)

	server := http.Server{
		Addr:    s.Addr,
		Handler: middleware(mux),
	}

	return server.ListenAndServe()
}

type Middleware func(http.Handler) http.HandlerFunc

func ChainMiddleware(middlewares ...Middleware) Middleware {
	return func(next http.Handler) http.HandlerFunc {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}

		return next.ServeHTTP
	}
}

func AuthMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if token := r.Header.Get("token"); token != "bearer" {
			http.Error(w, "Authentication failed. Invalid token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	}
}

func LoggerMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	}
}

func main() {
	server := NewServer(":8080")
	log.Println(server.Run())
}
