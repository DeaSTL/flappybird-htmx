package game

import (
	"sync"

	"github.com/deastl/flappybird-htmx/game/physics"
)

type PipeSet struct {
	ID             string
	X              int
	BottomY        int
	Y              int
	TopPieceHeight int
	Width          int
	Height         int
	TopCollider    physics.BoundingBox
	BottomCollider physics.BoundingBox
	PointCollider  physics.BoundingBox
	mut            sync.Mutex
	Visible        bool
}
