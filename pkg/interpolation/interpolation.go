package interpolation

type Interpolator interface {
	Interpolate(before, after []float64, gapLen int) []float64
}
