package main

import (
	"compress/flate"
	"context"
	"github.com/deastl/flappybird-htmx/db"
	"github.com/deastl/flappybird-htmx/game"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/v5/middleware"
	"log"
	"net/http"
	"strconv"
	"time"
)

func main() {

	server_state := game.ServerState{}

	dbq, err := db.NewConnection()

	if err != nil {
		log.Fatalf("Could not open database connection : %v", err)
	}

	ctx := context.Background()

	server_state.Dbq = dbq
	server_state.Ctx = ctx

	server_state.New()

	log.Println("Starting flappybird server")

	r := chi.NewRouter()

	compressor := middleware.NewCompressor(flate.BestSpeed)
	r.Use(middleware.Recoverer)
	r.Use(compressor.Handler)

	file_server := http.FileServer(http.Dir("./local/"))
	r.Handle("/local/*", http.StripPrefix("/local", file_server))

	// Report info to log

	go func() {
		for {
			time.Sleep(5 * time.Second)
			server_state.LogInfo()
		}
	}()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		err := server_state.PlayerEntered(w, r)

		if err != nil {
			log.Printf("Error when initializing player: %v", err)
		}
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("good"))
	})

	r.Get("/new-screen", func(w http.ResponseWriter, r *http.Request) {
		game_state, err := server_state.GetSessionGameState(r)

		if err != nil {
			log.Printf("Error in new-screen: %v", err)
		}

		target_fps_str := r.URL.Query().Get("target-fps")
		target_fps, err := strconv.ParseInt(target_fps_str, 10, 64)

		game_state.SetTargetFPS(int(target_fps))

		game_state.Mut.Lock()
		err = server_state.Templates.ExecuteTemplate(w, "templates/screen-frame.tmpl.html", game_state)
		game_state.Mut.Unlock()
		if err != nil {
			log.Printf("Error running screen-frame template %v", err)
		}
	})

	r.Get("/get-screen", func(w http.ResponseWriter, r *http.Request) {
		server_state.PlayerRequestedFrame(w, r)
	})

	r.Get("/get-dead-screen", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		server_state.Templates.ExecuteTemplate(w, "dead-screen.tmpl.html", []byte{})
	})

	r.Get("/empty", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(200)
		w.Write([]byte{})
	})

	r.Put("/jump-player", func(w http.ResponseWriter, r *http.Request) {
		server_state.PlayerJumped(w, r)
	})

	r.Get("/get-stats", func(w http.ResponseWriter, r *http.Request) {

		game_state, err := server_state.GetSessionGameState(r)
		if err != nil {
			log.Printf("Error in jump-player: %v", err)
		}

		game_state.Mut.Lock()
		err = server_state.Templates.ExecuteTemplate(w, "templates/stats.tmpl.html", game_state)
		game_state.Mut.Unlock()

	})

	err = http.ListenAndServe("0.0.0.0:3200", r)

	if err != nil {
		log.Printf("Error starting server: %v", err)
	}

}
