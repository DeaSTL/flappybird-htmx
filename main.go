package main

import (
	"log"
	"math/rand"
	"net/http"
	"text/template"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/v5/middleware"
)

type PipeSet struct {
	X       int
	BottomY int
	Y       int
	ID      string
	Visible bool
	Width   int
	Height  int
}

type Player struct {
	X       float32
	Y       float32
	Rot     float32
	Vel     float32
	Width   int
	Height  int
	Jumping bool
	Started bool
	Dead    bool
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
			Y:       vert_level,
			BottomY: vert_level + 300,
			X:       (i + s.pipe_starting_pos) * s.pipe_hor_offset,
			ID:      genID(12),
			Visible: true,
			Width:   255,
			Height:  135,
		}

		s.Pipes[new_pipe.ID] = new_pipe
	}
}

func (s *GameState) isColliding() bool {
	for _, pipe := range s.Pipes {
		if (s.Player.X > float32(pipe.X) && s.Player.X < float32(pipe.X)+64) &&
			(s.Player.Y < float32(pipe.Y+75) || s.Player.Y > float32(pipe.BottomY)) {
			return true
		}
	}
	return false
}

func (s *GameState) update() {
	if !s.Player.Dead {
		for key, _ := range s.Pipes {
			new_pipe := s.Pipes[key]
			new_pipe.X -= 70
			if new_pipe.X < -100 {
				// If it goes past the screen then send it to the back
				new_pipe.Visible = false
				furthest_pipe := s.getFurthestPipe()
				new_pipe.X = furthest_pipe.X + s.pipe_hor_offset
				// If it's outside the screen then we don't show it
			} else if new_pipe.X < 1500 && new_pipe.X > 0 {
				new_pipe.Visible = true
			}
			s.Pipes[key] = new_pipe
		}
	}

	if s.isColliding() {
		s.Player.Dead = true
	}
}

func newGameState() GameState {
	game_state := GameState{
		Player: Player{
			Y: 300,
			X: 100,
		},
		Pipes:             map[string]PipeSet{},
		PollRate:          "400ms",
		pipe_vert_offset:  300,
		pipe_count:        5,
		pipe_variation:    300,
		pipe_hor_offset:   300,
		pipe_starting_pos: 3,
	}
	game_state.genInitialPipes()

	return game_state

}

func main() {

	var game_state GameState

	log.Println("Starting flappybird server")

	templates, err := template.ParseGlob("./templates/*")

	if err != nil {
		log.Printf("Could not load templates %+v", err)
	}

	r := chi.NewRouter()

	//r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	file_server := http.FileServer(http.Dir("./local/"))
	r.Handle("/local/*", http.StripPrefix("/local", file_server))

	r.Put("/tick", func(w http.ResponseWriter, r *http.Request) {
		game_state.update()
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		game_state = newGameState()
		templates.ExecuteTemplate(w, "index.html", game_state)
	})

	r.Put("/jump-player", func(w http.ResponseWriter, r *http.Request) {
		game_state.Player.Started = true
		game_state.Player.Jumping = true
		w.WriteHeader(200)
	})

	r.Get("/get-dead", func(w http.ResponseWriter, r *http.Request) {
		if game_state.Player.Dead {
			templates.ExecuteTemplate(w, "dead-screen.html", []byte{})
		} else {
			w.WriteHeader(200)
		}
	})

	r.Get("/get-player", func(w http.ResponseWriter, r *http.Request) {

		game_state.Player.update()

		templates.ExecuteTemplate(w, "player.tmpl", game_state.Player)
	})

	r.Get("/get-pipe/{pipe_id}", func(w http.ResponseWriter, r *http.Request) {
		id_param := chi.URLParam(r, "pipe_id")
		if err != nil {
			log.Fatal("Could not parse pipe_id")
		}
		pipe := game_state.Pipes[id_param]

		templates.ExecuteTemplate(w, "pipe.tmpl", pipe)
	})

	http.ListenAndServe(":3200", r)

}
