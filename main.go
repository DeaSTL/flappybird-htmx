package main

import (
	"errors"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"text/template"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/v5/middleware"
)

type BoundingBox struct {
	X      float32
	Y      float32
	Width  float32
	Height float32
}

// AABB Collision
func (b *BoundingBox) isColliding(other *BoundingBox) bool {
	if b.X < other.X+other.Width &&
		b.X+b.Width > other.X &&
		b.Y < other.Y+other.Height &&
		b.Y+b.Height > other.Y {
		return true
	}
	return false
}

type PipeSet struct {
	X              int
	BottomY        int
	Y              int
	ID             string
	Visible        bool
	TopPieceHeight int
	Width          int
	Height         int
	TopCollider    BoundingBox
	BottomCollider BoundingBox
	mut            sync.Mutex
}

func genID(length int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz-_"

	id := make([]byte, length)

	for i := range id {
		id[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(id)
}

type ServerState struct {
	Templates  template.Template
	GameStates sync.Map
	mut        sync.Mutex
}

func (s *ServerState) loadTemplates() {
	s.mut.Lock()
	unsafe_templates, err := template.ParseGlob("./templates/*")

	if err != nil {
		log.Printf("Could not load templates: %v", err)
		os.Exit(-1)
	}

	s.Templates = *unsafe_templates

	s.mut.Unlock()

}

func (s *ServerState) getSessionGameState(r *http.Request) (GameState, error) {

	session, err := r.Cookie("session")

	session_id := session.Value

	if err != nil {
		return GameState{}, err
	}

	sync_state, ok := s.GameStates.Load(session_id)

	if !ok {
		return GameState{}, errors.New("Coud not load sync state from syncmap")
	}

	game_state := sync_state.(GameState)

	return game_state, nil
}

func (s *ServerState) setSessionGameState(r *http.Request, game_state GameState) error {

	session_id, err := r.Cookie("session")

	if err != nil {
		return err
	}

	s.GameStates.Store(session_id, game_state)

	return nil
}

func main() {

	server_state := ServerState{}

	new_templates, err := server_state.Templates.ParseGlob("./templates/*")

	if err != nil {

	}
	server_state.mut.Lock()
	server_state.Templates = *new_templates
	server_state.mut.Unlock()

	//var game_state GameState

	log.Println("Starting flappybird server")

	r := chi.NewRouter()

	//r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	file_server := http.FileServer(http.Dir("./local/"))
	r.Handle("/local/*", http.StripPrefix("/local", file_server))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {

		cookie := http.Cookie{
			Name:  "session",
			Value: genID(32),
		}

		http.SetCookie(w, &cookie)

		log.Printf("New user from: %s", cookie.Value)

		new_game_state := newGameState()

		server_state.GameStates.Store(cookie.Value, new_game_state)
		new_game_state.mut.Lock()

		err := server_state.Templates.ExecuteTemplate(w, "index.html", new_game_state)
		new_game_state.mut.Unlock()

		if err != nil {
			log.Printf("Could not render index template: %+v", err)
		}

		go func(session_id string) {
			delay := 30 * time.Millisecond
			for {
				sync_out, ok := server_state.GameStates.Load(session_id)

				game_state := sync_out.(GameState)

				if !ok {
					log.Print("Could not load game state in game loop")
					os.Exit(-1)
				}

				game_state.Player.update()
				game_state.update()
				time.Sleep(time.Duration(delay))

				server_state.GameStates.Store(session_id, game_state)
			}
		}(cookie.Value)

	})

	r.Get("/get-screen", func(w http.ResponseWriter, r *http.Request) {
		game_state, err := server_state.getSessionGameState(r)

		if err != nil {
			log.Printf("Error in get-screen: %v", err)
		}

		server_state.mut.Lock()
		err = server_state.Templates.ExecuteTemplate(w, "screen.html", game_state)
		server_state.mut.Unlock()

		if err != nil {
			log.Printf("Could not render index template: %+v", err)
		}
	})

	r.Put("/jump-player", func(w http.ResponseWriter, r *http.Request) {
		game_state, err := server_state.getSessionGameState(r)

		if err != nil {
			log.Printf("Error in jump-player: %v", err)
		}

		game_state.Player.Started = true
		game_state.Player.Jumping = true

		server_state.setSessionGameState(r, game_state)

		w.WriteHeader(200)
	})

	r.Get("/get-dead", func(w http.ResponseWriter, r *http.Request) {

		game_state, err := server_state.getSessionGameState(r)

		if err != nil {
			log.Printf("Error in get-dead: %v", err)
		}

		if game_state.Player.Dead {
			game_state.mut.Lock()
			err := server_state.Templates.ExecuteTemplate(w, "dead-screen.html", []byte{})
			server_state.mut.Unlock()

			if err != nil {
				log.Printf("Could not render index template: %+v", err)
				server_state.loadTemplates()
			}
		} else {
			w.WriteHeader(200)
		}
	})

	http.ListenAndServe(":3200", r)

}
