package protocol

import (
	"math"
	"reflect"
	"testing"
)

func TestEncode(t *testing.T) {
	tests := []struct {
		name     string
		x        uint16
		y        uint16
		color    uint8
		time     int64
		expected [8]byte
	}{
		{
			name:     "Basic encoding",
			x:        1000,
			y:        2000,
			color:    10,
			time:     referenceTime + 3600000, // 1 hour after reference
			expected: [8]byte{0b10000011, 0b11101000, 0b10000111, 0b11010000, 0x00, 0x36, 0xEE, 0x80},
		},
		{
			name:     "Min values",
			x:        0,
			y:        0,
			color:    0,
			time:     referenceTime,
			expected: [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cell := Cell{
				X:     tt.x,
				Y:     tt.y,
				Color: tt.color,
				Time:  tt.time,
			}
			result := cell.Encode()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDecode(t *testing.T) {
	tests := []struct {
		name  string
		input [8]byte
		x     uint16
		y     uint16
		color uint8
		time  int64
	}{
		{
			name:  "Basic decoding",
			input: [8]byte{0b10000011, 0b11101000, 0b10000111, 0b11010000, 0x00, 0x36, 0xEE, 0x80},
			x:     1000,
			y:     2000,
			color: 10,
			time:  referenceTime + 3600000, // 1 hour after reference
		},
		{
			name:  "Max values",
			input: [8]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			x:     0x3FFF, // 16383
			y:     0x3FFF, // 16383
			color: 15,     // 0b1111
			time:  referenceTime + math.MaxUint32,
		},
		{
			name:  "Min values",
			input: [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			x:     0,
			y:     0,
			color: 0,
			time:  referenceTime,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			cell := Decode(tt.input)
			if cell.X != tt.x || cell.Y != tt.y || cell.Color != tt.color || cell.Time != tt.time {
				t.Errorf("want %v, got %v", tt, cell)
			}
		})
	}
}

func TestEncodeDecode(t *testing.T) {

	cell := Cell{
		X:     uint16(12345),
		Y:     uint16(9321),
		Color: uint8(7),
		Time:  int64(referenceTime + 86400000),
	}

	encoded := cell.Encode()
	cellDecoded := Decode(encoded)

	if reflect.DeepEqual(cell, cellDecoded) {
		t.Errorf("want %v, got %v", cell, cellDecoded)
	}
}

func BenchmarkEncode(b *testing.B) {
	cell := Cell{
		X:     uint16(12345),
		Y:     uint16(9321),
		Color: uint8(7),
		Time:  int64(referenceTime + 86400000),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cell.Encode()
	}
}

func BenchmarkDecode(b *testing.B) {
	encoded := [8]byte{0x30, 0x39, 0xD4, 0x31, 0x00, 0x51, 0x61, 0x80}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Decode(encoded)
	}
}

func BenchmarkEncodeDecode(b *testing.B) {
	cell := Cell{
		X:     uint16(12345),
		Y:     uint16(9321),
		Color: uint8(7),
		Time:  int64(referenceTime + 86400000),
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encoded := cell.Encode()
		Decode(encoded)
	}
}
