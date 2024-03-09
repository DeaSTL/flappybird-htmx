package physics

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
func (b *BoundingBox) IsColliding(other *BoundingBox) bool {
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
