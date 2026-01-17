package interpolation

type dummy struct{}

func NewDummy() Interpolator {
	return &dummy{}
}

func (d *dummy) Interpolate(before, after []float64, gapLen int) []float64 {
	result := make([]float64, gapLen)
	if len(before) == 0 || len(after) == 0 {
		return result
	}
	v0 := before[len(before)-1]
	v1 := after[0]
	for i := 0; i < gapLen; i++ {
		t := float64(i+1) / float64(gapLen+1)
		result[i] = (1-t)*v0 + t*v1
	}
	return result
}
