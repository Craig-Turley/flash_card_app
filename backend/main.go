package main

import (
	"log"
	"net/http"
)

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

func createFlashCardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Bruh", http.StatusMethodNotAllowed)
	}
	w.Write([]byte(r.Header.Get("token") + "\n"))
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
		AuthMiddleware,
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
