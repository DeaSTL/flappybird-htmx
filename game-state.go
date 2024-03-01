package main

import (
	"log"
	"math/rand"
	"sync"
)

type GameState struct {
	Player            Player
	Pipes             map[string]*PipeSet
	PollRate          string
	pipe_hor_offset   int
	pipe_vert_offset  int
	pipe_starting_pos int
	pipe_variation    int
	pipe_count        int
	mut               sync.Mutex
}

func (s *GameState) getFurthestPipe() *PipeSet {
	furthest := &PipeSet{}
	for _, pipe := range s.Pipes {
		if pipe.X > furthest.X {
			furthest = pipe
		}
	}

	return furthest
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

		s.Pipes[new_pipe.ID] = &new_pipe
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
	s.mut.Lock()
	defer s.mut.Unlock()
	if !s.Player.Dead {
		for key := range s.Pipes {
			new_pipe := s.Pipes[key]
			new_pipe.X -= 20
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
			new_pipe.TopCollider.Y = float32(new_pipe.Y - 5110) // Not even I know how I got this value
			new_pipe.BottomCollider.X = float32(new_pipe.X)
			new_pipe.BottomCollider.Y = float32(new_pipe.BottomY)

			s.Pipes[key] = new_pipe

		}
	}

	if s.isColliding() {
		s.Player.Dead = true
	}
}

func newGameState() *GameState {
	game_state := GameState{
		Player: Player{
			Y:      300,
			X:      100,
			Width:  50,
			Height: 32,
		},
		Pipes:             map[string]*PipeSet{},
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

	return &game_state
}
