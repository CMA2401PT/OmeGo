package define

// Pos holds the position of a block. The position is represented of an array with an x, y and z value,
// where the y value is positive.
type Pos [3]int

// X returns the X coordinate of the block position.
func (p Pos) X() int {
	return p[0]
}

// Y returns the Y coordinate of the block position.
func (p Pos) Y() int {
	return p[1]
}

// Z returns the Z coordinate of the block position.
func (p Pos) Z() int {
	return p[2]
}

// Add adds two block positions together and returns a new one with the combined values.
func (p Pos) Add(pos Pos) Pos {
	return Pos{p[0] + pos[0], p[1] + pos[1], p[2] + pos[2]}
}

// Subtract subtracts two block positions together and returns a new one with the combined values.
func (p Pos) Subtract(pos Pos) Pos {
	return Pos{p[0] - pos[0], p[1] - pos[1], p[2] - pos[2]}
}
