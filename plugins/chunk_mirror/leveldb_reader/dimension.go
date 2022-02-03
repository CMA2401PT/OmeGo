package main

import "time"

var (
	// Overworld is the Dimension implementation of a normal overworld. It has a blue sky under normal circumstances and
	// has a sun, clouds, stars and a moon. Overworld has a building range of [-64, 320].
	Overworld overworld
	// Nether is a Dimension implementation with a lower base light level and a darker sky without sun/moon. It has a
	// building range of [0, 256].
	Nether nether
	// End is a Dimension implementation with a dark sky. It has a building range of [0, 256].
	End end
)

// Range represents the height range of a Dimension in blocks. The first value of the Range holds the minimum Y value,
// the second value holds the maximum Y value.
type Range [2]int

// Min returns the minimum Y value of a Range. It is equivalent to Range[0].
func (r Range) Min() int {
	return r[0]
}

// Max returns the maximum Y value of a Range. It is equivalent to Range[1].
func (r Range) Max() int {
	return r[1]
}

// Height returns the total height of the Range, the difference between Max and Min.
func (r Range) Height() int {
	return r[1] - r[0]
}

type (
	// Dimension is a dimension of a World. It influences a variety of properties of a World such as the building range,
	// the sky colour and the behaviour of liquid blocks.
	Dimension interface {
		Range() Range
		EncodeDimension() int
		WaterEvaporates() bool
		LavaSpreadDuration() time.Duration
		WeatherCycle() bool
		TimeCycle() bool
	}
	overworld struct{}
	nether    struct{}
	end       struct{}
)

func (overworld) Range() Range                      { return Range{-64, 320} }
func (overworld) EncodeDimension() int              { return 0 }
func (overworld) WaterEvaporates() bool             { return false }
func (overworld) LavaSpreadDuration() time.Duration { return time.Second * 3 / 2 }
func (overworld) WeatherCycle() bool                { return true }
func (overworld) TimeCycle() bool                   { return true }
func (overworld) String() string                    { return "Overworld" }
