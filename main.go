package main

import (
	"bytes"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"os"
	"sync"
	"text/template"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/v5/middleware"
)

//go:embed templates/*
var static embed.FS

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
	ID             string
	X              int
	BottomY        int
	Y              int
	TopPieceHeight int
	Width          int
	Height         int
	TopCollider    BoundingBox
	BottomCollider BoundingBox
	mut            sync.Mutex
	Visible        bool
}

func genID(length int) string {
	idBytes := make([]byte, length)
	rand.Read(idBytes)
	buf := bytes.NewBuffer([]byte{})
	base64.NewEncoder(base64.RawURLEncoding.Strict(), buf)
	return string(buf.String())
}

type ServerState struct {
	GameStates sync.Map
	mut        sync.Mutex
}

func (s *ServerState) getSessionGameState(r *http.Request) (*GameState, error) {

	session, err := r.Cookie("session")

	session_id := session.Value

	if err != nil {
		return &GameState{}, err
	}

	sync_state, ok := s.GameStates.Load(session_id)
	if !ok {
		return &GameState{}, errors.New("coud not load sync state from syncmap")
	}

	game_state := sync_state.(*GameState)

	return game_state, nil
}

func (s *ServerState) setSessionGameState(r *http.Request, game_state *GameState) error {

	session_id, err := r.Cookie("session")

	if err != nil {
		return err
	}

	s.GameStates.Store(session_id, game_state)

	return nil
}

var boundingBoxTempl *template.Template
var deadScrentTempl *template.Template
var indexTempl *template.Template
var pipeTempl *template.Template
var playerTempl *template.Template
var screenTempl *template.Template
var statsTempl *template.Template

func init() {

	x := map[string]*template.Template{
		"bounding-box": boundingBoxTempl,
		"dead-screen":  deadScrentTempl,
		"index":        indexTempl,
		"pipe":         pipeTempl,
		"player":       playerTempl,
		"screen":       screenTempl,
		"stats":        statsTempl,
	}
	for f, t := range x {
		fileContents, err := static.ReadFile(f + ".html")
		if err != nil {
			panic(err.Error())
		}
		_, err = t.Parse(string(fileContents))
		if err != nil {
			panic(err.Error())
		}

	}
}

func main() {

	server_state := ServerState{}

	log.Println("Starting flappybird server")

	r := chi.NewRouter()

	//r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	file_server := http.FileServer(http.Dir("./local/"))
	r.Handle("/local/*", http.StripPrefix("/local", file_server))

	r.Get("/", func(w http.ResponseWriter, _ *http.Request) {

		cookie := http.Cookie{
			Name:  "session",
			Value: genID(32),
		}

		http.SetCookie(w, &cookie)

		log.Printf("New user from: %s", cookie.Value)

		new_game_state := newGameState()

		server_state.GameStates.Store(cookie.Value, new_game_state)

		err := indexTempl.Execute(w, new_game_state)
		if err != nil {
			log.Printf("Could not render index template: %+v", err)
		}

		go func(session_id string) {
			delay := 30 * time.Millisecond
			for {
				sync_out, ok := server_state.GameStates.Load(session_id)

				game_state := sync_out.(*GameState)

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

		screenTempl.Execute(w, game_state)

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
			err := deadScrentTempl.Execute(w, []byte{})

			if err != nil {
				log.Printf("Could not render index template: %+v", err)
				http.Error(w, "", http.StatusInternalServerError)
				return
			}
		} else {
			w.WriteHeader(200)
		}
	})

	http.ListenAndServe(":3200", r)

}
