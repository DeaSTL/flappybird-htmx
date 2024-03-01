package main

import (
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

type Player struct {
	X        float32
	Y        float32
	Rot      float32
	Vel      float32
	Width    int
	Height   int
	Jumping  bool
	Started  bool
	Dead     bool
	Collider BoundingBox
	mut      sync.Mutex
}

func (s *Player) update() {
	if s.Started {
		s.Vel += 0.011

		if s.Jumping && !s.Dead {
			s.Vel -= 0.19
			s.Jumping = false
		}

		s.Y += s.Vel * 20

		if s.Y > 1200 {
			s.Dead = true
		}
	}

	//Changes the players displayed rotation
	if s.Vel > -0.1 && s.Vel < 0.1 {
		s.Rot = 0
	}
	if s.Vel <= -0.1 {
		s.Rot = -0.1
	}
	if s.Vel >= 0.1 {
		s.Rot = 0.1
	}

	s.Collider.X = s.X
	s.Collider.Y = s.Y
}

type GameState struct {
	Player            Player
	Pipes             map[string]PipeSet
	PollRate          string
	pipe_hor_offset   int
	pipe_vert_offset  int
	pipe_starting_pos int
	pipe_variation    int
	pipe_count        int
	Templates         template.Template
	mut               sync.Mutex
}

func (s *GameState) getFurthestPipe() PipeSet {
	var furthest PipeSet
	for _, pipe := range s.Pipes {
		if pipe.X > furthest.X {
			furthest = pipe
		}
	}

	return furthest
}

func genID(length int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz-_"

	id := make([]byte, length)

	for i := range id {
		id[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(id)
}

func (s *GameState) genInitialPipes() {
	num_pipes := s.pipe_count
	for i := 0; i < num_pipes; i++ {
		vert_level := rand.Intn(s.pipe_variation)

		new_pipe := PipeSet{
			Y:              vert_level,
			BottomY:        vert_level + 300,
			X:              i * (s.pipe_starting_pos + s.pipe_hor_offset),
			ID:             genID(12),
			Visible:        true,
			Width:          255,
			TopPieceHeight: 135,
			Height:         5000,
		}

		new_pipe.Height += new_pipe.TopPieceHeight

		new_pipe.TopCollider.X = float32(new_pipe.X)
		new_pipe.BottomCollider.X = float32(new_pipe.X)

		new_pipe.BottomCollider.Y = float32(new_pipe.BottomY) + 100
		new_pipe.TopCollider.Y = float32(new_pipe.Y)

		new_pipe.TopCollider.Width = float32(new_pipe.Width / 4)
		new_pipe.BottomCollider.Width = float32(new_pipe.Width / 4)

		new_pipe.TopCollider.Height = float32(new_pipe.Height)
		new_pipe.BottomCollider.Height = float32(new_pipe.Height)

		log.Printf("pipe ID: %s Top Collider: %+v", new_pipe.ID, new_pipe.TopCollider)
		log.Printf("pipe ID: %s Bottom Collider: %+v", new_pipe.ID, new_pipe.BottomCollider)

		s.Pipes[new_pipe.ID] = new_pipe
	}
}

func (s *GameState) isColliding() bool {
	for _, pipe := range s.Pipes {
		if pipe.BottomCollider.isColliding(&s.Player.Collider) ||
			pipe.TopCollider.isColliding(&s.Player.Collider) {
			return true
		}
	}
	return false
}

func (s *GameState) update() {
	if !s.Player.Dead {
		for key, _ := range s.Pipes {
			new_pipe := s.Pipes[key]
			new_pipe.X -= 5
			if new_pipe.X < -100 {
				// If it goes past the screen then send it to the back
				new_pipe.Visible = false
				furthest_pipe := s.getFurthestPipe()
				new_pipe.X = furthest_pipe.X + s.pipe_hor_offset
				// If it's outside the screen then we don't show it
			} else if new_pipe.X < 1500 && new_pipe.X > 0 {
				new_pipe.Visible = true
			}

			new_pipe.TopCollider.X = float32(new_pipe.X)
			new_pipe.TopCollider.Y = float32(new_pipe.Y - 5110)
			new_pipe.BottomCollider.X = float32(new_pipe.X)
			new_pipe.BottomCollider.Y = float32(new_pipe.BottomY)

			s.mut.Lock()
			s.Pipes[key] = new_pipe
			s.mut.Unlock()

		}
	}

	if s.isColliding() {
		s.Player.Dead = true
	}
}

func (s *GameState) loadTemplates() {
	s.mut.Lock()
	unsafe_templates, err := template.ParseGlob("./templates/*")

	if err != nil {
		log.Printf("Could not load templates: %v", err)
		os.Exit(-1)
	}

	s.Templates = *unsafe_templates

	s.mut.Unlock()

}
func newGameState() GameState {
	game_state := GameState{
		Player: Player{
			Y:      300,
			X:      100,
			Width:  50,
			Height: 32,
		},
		Pipes:             map[string]PipeSet{},
		PollRate:          "30ms",
		pipe_vert_offset:  300,
		pipe_count:        8,
		pipe_variation:    300,
		pipe_hor_offset:   300,
		pipe_starting_pos: 300,
	}
	game_state.Player.Collider = BoundingBox{
		X:      game_state.Player.X,
		Y:      game_state.Player.Y,
		Width:  float32(game_state.Player.Width),
		Height: float32(game_state.Player.Height),
	}
	game_state.genInitialPipes()

	return game_state
}

func main() {

	var game_state GameState

	log.Println("Starting flappybird server")

	go func() {
		delay := 30 * time.Millisecond
		for {
			game_state.Player.update()
			game_state.update()
			time.Sleep(time.Duration(delay))
		}
	}()

	r := chi.NewRouter()

	//r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	file_server := http.FileServer(http.Dir("./local/"))
	r.Handle("/local/*", http.StripPrefix("/local", file_server))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		game_state = newGameState()

		game_state.loadTemplates()

		game_state.mut.Lock()

		err := game_state.Templates.ExecuteTemplate(w, "index.html", game_state)
		game_state.mut.Unlock()

		if err != nil {
			log.Printf("Could not render index template: %+v", err)
			game_state.loadTemplates()
		}
	})

	r.Get("/get-screen", func(w http.ResponseWriter, r *http.Request) {
		game_state.mut.Lock()
		err := game_state.Templates.ExecuteTemplate(w, "screen.html", game_state)
		game_state.mut.Unlock()

		if err != nil {
			log.Printf("Could not render index template: %+v", err)
			game_state.loadTemplates()
		}
	})

	r.Put("/jump-player", func(w http.ResponseWriter, r *http.Request) {
		game_state.Player.Started = true
		game_state.Player.Jumping = true
		w.WriteHeader(200)
	})

	r.Get("/get-dead", func(w http.ResponseWriter, r *http.Request) {
		if game_state.Player.Dead {
			game_state.mut.Lock()
			err := game_state.Templates.ExecuteTemplate(w, "dead-screen.html", []byte{})
			game_state.mut.Unlock()

			if err != nil {
				log.Printf("Could not render index template: %+v", err)
				game_state.loadTemplates()
			}
		} else {
			w.WriteHeader(200)
		}
	})

	r.Get("/get-player", func(w http.ResponseWriter, r *http.Request) {
		game_state.mut.Lock()
		err := game_state.Templates.ExecuteTemplate(w, "player.tmpl", game_state.Player)
		game_state.mut.Unlock()

		if err != nil {
			log.Printf("Could not render index template: %+v", err)
			game_state.loadTemplates()
		}
	})

	http.ListenAndServe(":3200", r)

}
