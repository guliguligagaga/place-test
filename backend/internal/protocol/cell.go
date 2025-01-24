package protocol

import (
	"encoding/binary"
)

const referenceTime = 1704067200000

type Cell struct {
	X, Y  uint16
	Color uint8
	Time  int64
}

func (c *Cell) Encode() [8]byte {
	var encoded [8]byte
	combinedX := c.X | uint16(c.Color&0b1100)<<12
	binary.BigEndian.PutUint16(encoded[0:2], combinedX)
	combinedY := c.Y | uint16(c.Color&0b0011)<<14
	binary.BigEndian.PutUint16(encoded[2:4], combinedY)
	compactTimestamp := uint32(c.Time - referenceTime)
	binary.BigEndian.PutUint32(encoded[4:], compactTimestamp)

	return encoded
}

func Decode(encoded [8]byte) *Cell {
	combinedX := binary.BigEndian.Uint16(encoded[0:2])
	x := combinedX & 0x3FFF
	color := uint8((combinedX >> 12) & 0b1100)

	combinedY := binary.BigEndian.Uint16(encoded[2:4])
	y := combinedY & 0x3FFF
	color |= uint8((combinedY >> 14) & 0b0011)

	millisDiff := binary.BigEndian.Uint32(encoded[4:])
	time := referenceTime + int64(millisDiff)

	return &Cell{
		X:     x,
		Y:     y,
		Color: color,
		Time:  time,
	}
}
