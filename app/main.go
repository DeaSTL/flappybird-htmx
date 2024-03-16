package main

import (
	"compress/flate"
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/deastl/flappybird-htmx/db"
	"github.com/deastl/flappybird-htmx/game"
	mid "github.com/deastl/flappybird-htmx/middlware"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/v5/middleware"
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
	r.Use(mid.InitializeUserSession(&server_state))

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
			http.Error(w, "Error when initializing player: "+err.Error(), 500)
			return
		}
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("good"))
	})

	r.Post("/update-fps", func(w http.ResponseWriter, r *http.Request) {
		game_state, err := server_state.GetSessionGameState(r)

		if err != nil {
			http.Error(w, "Error in new-screen: "+err.Error(), 500)
			return
		}

		target_fps_str := r.URL.Query().Get("value")
		target_fps, err := strconv.ParseInt(target_fps_str, 10, 64)

		game_state.SetTargetFPS(int(target_fps))

		game_state.Mut.Lock()
		err = server_state.Templates.ExecuteTemplate(w, "templates/screen-frame.tmpl.html", game_state)
		game_state.Mut.Unlock()
		if err != nil {
			http.Error(w, "Error running screen-frame template", 500)
			return
		}
	})

	r.Get("/get-screen", func(w http.ResponseWriter, r *http.Request) {
		err := server_state.PlayerRequestedFrame(w, r)

		if err != nil {
			http.Error(w, "Error running get-screen: "+err.Error(), 500)
			return
		}

	})

	r.Get("/get-dead-screen", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		err := server_state.Templates.ExecuteTemplate(w, "dead-screen.tmpl.html", []byte{})

		if err != nil {
			http.Error(w, "Error in get-dead-screen: "+err.Error(), 500)
		}
	})

	r.Get("/empty", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(200)
		w.Write([]byte{})
	})

	r.Put("/jump-player", func(w http.ResponseWriter, r *http.Request) {
		err := server_state.PlayerJumped(w, r)

		if err != nil {
			http.Error(w, "Error jumping: "+err.Error(), 500)
			return
		}
	})

	r.Get("/get-stats", func(w http.ResponseWriter, r *http.Request) {

		game_state, err := server_state.GetSessionGameState(r)
		if err != nil {
			http.Error(w, "Error in jump-player: "+err.Error(), 500)
			return
		}

		game_state.Mut.Lock()
		err = server_state.Templates.ExecuteTemplate(w, "templates/stats.tmpl.html", game_state)
		game_state.Mut.Unlock()

		if err != nil {
			http.Error(w, "Error running template in get-stats: "+err.Error(), 500)
			return
		}
	})

	err = http.ListenAndServe("0.0.0.0:3200", r)

	if err != nil {
		log.Printf("Error starting server: %v", err)
	}

}
