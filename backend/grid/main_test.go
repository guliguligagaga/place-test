package grid

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCalculateOffset(t *testing.T) {
	tests := []struct {
		name           string
		x, y           uint16
		expectedOffset int64
	}{
		{
			name:           "Upper nibble",
			x:              0,
			y:              0,
			expectedOffset: 0,
		},
		{
			name:           "Lower nibble",
			x:              1,
			y:              0,
			expectedOffset: 4,
		},
		{
			name:           "Different row",
			x:              0,
			y:              1,
			expectedOffset: 400, // (1 * 100 + 0) / 2 * 8
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			offset := calculateOffset(tt.y, tt.x, Size)
			assert.Equal(t, tt.expectedOffset, offset)
		})
	}
}
