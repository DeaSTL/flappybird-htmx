package main

import "sync"

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
