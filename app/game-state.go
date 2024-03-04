package main

import (
	"log"
	"math/rand"
	"sync"
	"time"
)

type GameState struct {
	Player                 Player
	Pipes                  map[string]*PipeSet
	PollRate               string
	DebugMode              bool
	Points                 int
	BackgroundOffset       int
	BackgroundGroundOffset int
	ClientAlive            bool
	ClientAliveTimer       *time.Timer
	FrameTimer             *time.Timer
	FPS                    int
	FrameCount             int
	pipe_hor_offset        int
	pipe_vert_offset       int
	pipe_starting_pos      int
	pipe_variation         int
	pipe_count             int
	in_point_collider      bool
	mut                    sync.Mutex
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
	for i := 1; i < num_pipes+1; i++ {
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
		new_pipe.PointCollider.X = float32(new_pipe.X)

		new_pipe.BottomCollider.Y = float32(new_pipe.BottomY)
		new_pipe.TopCollider.Y = float32(new_pipe.Y)

		new_pipe.TopCollider.Width = float32(new_pipe.Width / 4)
		new_pipe.BottomCollider.Width = float32(new_pipe.Width / 4)

		new_pipe.TopCollider.Height = float32(new_pipe.Height)
		new_pipe.BottomCollider.Height = float32(new_pipe.Height)

		on_point_collected := func(name string) {
			s.Points++
		}

		new_pipe.PointCollider.OnLeave = on_point_collected

		log.Printf("pipe ID: %s Top Collider: %+v", new_pipe.ID, new_pipe.TopCollider)
		log.Printf("pipe ID: %s Bottom Collider: %+v", new_pipe.ID, new_pipe.BottomCollider)
		log.Printf("pipe ID: %s Point Collider: %+v", new_pipe.ID, new_pipe.PointCollider)

		s.Pipes[new_pipe.ID] = &new_pipe
	}
}

func (s *GameState) isColliding() bool {
	for _, pipe := range s.Pipes {
		if pipe.BottomCollider.isColliding(&s.Player.Collider) ||
			pipe.TopCollider.isColliding(&s.Player.Collider) {
			return true
		}
		pipe.PointCollider.isColliding(&s.Player.Collider)
	}
	return false
}

func (s *GameState) update() {
	s.mut.Lock()
	defer s.mut.Unlock()

	if !s.Player.Dead && s.Player.Started {
		s.BackgroundOffset -= 1
		s.BackgroundGroundOffset -= 15
		for key := range s.Pipes {
			vert_level := rand.Intn(s.pipe_variation)
			gap_level := rand.Intn(100)

			bottom_y_offset := vert_level + 150 + gap_level
			new_pipe := s.Pipes[key]
			new_pipe.X -= 15
			if new_pipe.X < -100 {
				// If it goes past the screen then send it to the back
				new_pipe.Visible = false
				new_pipe.Y = vert_level
				new_pipe.BottomY = bottom_y_offset
				furthest_pipe := s.getFurthestPipe()
				new_pipe.X = furthest_pipe.X + (s.pipe_hor_offset * 2)
				// If it's outside the screen then we don't show it
			} else if new_pipe.X < 1500 && new_pipe.X > 0 {
				new_pipe.Visible = true
			}

			new_pipe.TopCollider.X = float32(new_pipe.X)
			new_pipe.TopCollider.Y = float32(new_pipe.Y - 5110) // Not even I know how I got this value

			new_pipe.PointCollider.X = float32(new_pipe.X)
			new_pipe.PointCollider.Y = float32(new_pipe.Y)
			new_pipe.PointCollider.Width = float32(new_pipe.Width / 4)
			new_pipe.PointCollider.Height = float32(new_pipe.BottomY - new_pipe.Y)

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
		DebugMode:         false,
		ClientAlive:       true,
		Pipes:             map[string]*PipeSet{},
		PollRate:          "35ms",
		pipe_vert_offset:  400,
		pipe_count:        8,
		pipe_variation:    250,
		pipe_hor_offset:   300,
		pipe_starting_pos: 300,
	}

	log.Printf("%+v", &game_state.Player)
	game_state.Player.Collider = BoundingBox{
		X:      game_state.Player.X,
		Y:      game_state.Player.Y,
		Width:  float32(game_state.Player.Width),
		Height: float32(game_state.Player.Height),
	}
	game_state.genInitialPipes()

	return &game_state
}
