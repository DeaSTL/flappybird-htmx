package main

import (
	"compress/flate"
	"embed"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/v5/middleware"
)

//go:embed templates/*
var static embed.FS

type BoundingBox struct {
	X         float32
	Y         float32
	Width     float32
	Height    float32
	Name      string
	Colliding bool
	OnEnter   func(object_name string)
	OnLeave   func(object_name string)
}

// AABB Collision
func (b *BoundingBox) isColliding(other *BoundingBox) bool {
	if b.X < other.X+other.Width &&
		b.X+b.Width > other.X &&
		b.Y < other.Y+other.Height &&
		b.Y+b.Height > other.Y {
		if !b.Colliding {
			if b.OnEnter != nil {
				b.OnEnter(b.Name)
			}
		}
		b.Colliding = true
		return true
	}
	if b.Colliding {
		if b.OnLeave != nil {
			b.OnLeave(b.Name)
		}
		b.Colliding = false
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
	PointCollider  BoundingBox
	mut            sync.Mutex
	Visible        bool
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
	GameStates sync.Map
	mut        sync.Mutex
}

func (s *ServerState) getSessionGameState(r *http.Request) (*GameState, error) {
	s.mut.Lock()
	defer s.mut.Unlock()
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

	s.GameStates.Store(session_id.Value, game_state)

	return nil
}

var megaTempl = template.New("")

func minifyTemplate(templ_text string) string {

	minified_text := templ_text

	minified_text = strings.ReplaceAll(minified_text, "\n", "")

	minified_text = strings.ReplaceAll(minified_text, ";  ", ";")

	return minified_text
}

func init() {

	x := []string{
		"templates/bounding-box.tmpl.css",
		"templates/dead-screen.tmpl.html",
		"templates/index.tmpl.html",
		"templates/pipe.tmpl.css",
		"templates/player.tmpl.css",
		"templates/screen.tmpl.css",
		"templates/stats.tmpl.html",
	}
	for _, f := range x {
		fileContents, err := static.ReadFile(f)
		if err != nil {
			panic(err.Error())
		}
		_, err = megaTempl.New(f).Parse(minifyTemplate(string(fileContents)))
		if err != nil {
			panic(err.Error())
		}

	}
}

func logInfo(server_state *ServerState) {
	log.Println("---------------------------------")
	log.Println("       Connected Clients         ")
	server_state.GameStates.Range(func(key any, value any) bool {
		id := key.(string)
		game_state := value.(*GameState)

		log.Printf(
			"ID: %s Score: %v PlayerAlive: %t FPS: %d \n",
			id,
			game_state.Points,
			!game_state.Player.Dead,
			game_state.FPS,
		)
		return true
	})
}

func main() {

	server_state := ServerState{}

	log.Println("Starting flappybird server")

	r := chi.NewRouter()

	//r.Use(middleware.Logger)

	compressor := middleware.NewCompressor(flate.DefaultCompression)
	r.Use(middleware.Recoverer)
	r.Use(compressor.Handler)

	file_server := http.FileServer(http.Dir("./local/"))
	r.Handle("/local/*", http.StripPrefix("/local", file_server))

	// Report info to log

	go func() {
		for {
			time.Sleep(5 * time.Second)
			logInfo(&server_state)
		}
	}()

	r.Get("/", func(w http.ResponseWriter, _ *http.Request) {

		cookie := http.Cookie{
			Name:  "session",
			Value: genID(32),
		}

		http.SetCookie(w, &cookie)

		log.Printf("New user from: %s", cookie.Value)

		new_game_state := newGameState()

		server_state.GameStates.Store(cookie.Value, new_game_state)

		err := megaTempl.ExecuteTemplate(w, "templates/index.tmpl.html", new_game_state)
		if err != nil {
			log.Printf("Could not render index template: %+v", err)
		}

		go func(session_id string) {
			delay := 30 * time.Millisecond
			for {
				sync_out, ok := server_state.GameStates.Load(session_id)

				game_state := sync_out.(*GameState)

				if game_state.ClientAliveTimer == nil {
					game_state.ClientAliveTimer = time.NewTimer(1 * time.Minute)
				}

				select {
				case <-game_state.ClientAliveTimer.C:
					game_state.ClientAlive = false
					server_state.GameStates.Delete(session_id)
					return
				default:
				}

				if !ok {
					log.Print("Could not load game state in game loop")
					os.Exit(-1)
				}
				game_state.Player.update()
				game_state.update()
				server_state.GameStates.Store(session_id, game_state)

				time.Sleep(time.Duration(delay))
			}
		}(cookie.Value)

	})

	r.Get("/get-screen", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		game_state, err := server_state.getSessionGameState(r)

		if game_state.ClientAliveTimer == nil || game_state.FrameTimer == nil {
			game_state.ClientAliveTimer = time.NewTimer(1 * time.Minute)
			game_state.FrameTimer = time.NewTimer(30 * time.Second)
		}

		select {
		case <-game_state.FrameTimer.C:
			game_state.FPS = game_state.FrameCount / 30
			game_state.FrameCount = 0
			game_state.FrameTimer.Reset(30 * time.Second)
		default:
			game_state.FrameCount++
		}

		game_state.ClientAliveTimer.Reset(1 * time.Minute)

		if err != nil {
			log.Printf("Error in get-screen: %v", err)
		}

		game_state.mut.Lock()
		err = megaTempl.ExecuteTemplate(w, "templates/screen.tmpl.css", game_state)
		game_state.mut.Unlock()

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

	r.Get("/get-stats", func(w http.ResponseWriter, r *http.Request) {

		game_state, err := server_state.getSessionGameState(r)
		if err != nil {
			log.Printf("Error in jump-player: %v", err)
		}

		game_state.mut.Lock()
		err = megaTempl.ExecuteTemplate(w, "templates/stats.tmpl.html", game_state)
		game_state.mut.Unlock()

	})

	r.Get("/get-dead", func(w http.ResponseWriter, r *http.Request) {
		game_state, err := server_state.getSessionGameState(r)

		if err != nil {
			log.Printf("Error in get-dead: %v", err)
		}

		if game_state.Player.Dead {
			game_state.mut.Lock()
			err := megaTempl.ExecuteTemplate(w, "templates/dead-screen.tmpl.html", game_state)
			game_state.mut.Unlock()

			if err != nil {
				log.Printf("Could not render index template: %+v", err)
				http.Error(w, "", http.StatusInternalServerError)
				return
			}
		} else {
			w.WriteHeader(200)
		}
	})

	err := http.ListenAndServe("0.0.0.0:3200", r)

	if err != nil {
		log.Printf("Error starting server: %v", err)
	}

}
