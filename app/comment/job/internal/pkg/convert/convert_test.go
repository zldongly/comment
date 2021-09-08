package convert

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInt64sToString(t *testing.T) {
	tests := []struct {
		give []int64
		want string
	}{
		{
			give: []int64{},
			want: "",
		},
		{
			give: []int64{1},
			want: "1",
		},
		{
			give: []int64{1, 2, 3},
			want: "1,2,3",
		},
	}

	for _, tt := range tests {
		result := Int64sToString(tt.give)
		assert.Equal(t, tt.want, result)
	}
}

func TestStringToInt64s(t *testing.T) {
	tests := []struct {
		give string
		want []int64
	}{
		{
			give: "",
			want: make([]int64, 0),
		},
		{
			give: "1",
			want: []int64{1},
		},
		{
			give: "1,2",
			want: []int64{1, 2},
		},
	}

	for _, tt := range tests {
		result := StringToInt64s(tt.give)
		assert.Equal(t, tt.want, result)
	}
}
