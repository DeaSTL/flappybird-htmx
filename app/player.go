package main

import (
	"log"
	"sync"
)

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

func NewPlayer(player *Player) {
	player.Collider.OnEnter = func(object_name string) {
		log.Println("OnEnter")
	}
	player.Collider.OnLeave = func(object_name string) {
		log.Println("OnLeave")
	}
}

func (s *Player) update() {
	s.mut.Lock()
	defer s.mut.Unlock()
	if s.Started {
		s.Vel += 0.019

		if s.Jumping && !s.Dead {
			s.Vel = -0.19
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
