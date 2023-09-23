package util

const (

	// DefaultEMASmoothing is a common smoothing constant for the EMA algorithm.
	DefaultWindowSize = 5
)

// EMA represents the state of an Exponential Moving Average (EMA).
type EMA struct {
	Init     bool
	Avg      float64
	Last     int64
	Constant float64
}

// NewEMA creates a new EMA data structure.
func NewEMA(size uint) *EMA {
	if size == 0 {
		size = DefaultWindowSize
	}
	ema := &EMA{
		Constant: 1 / (1 + float64(size)),
	}

	return ema
}

// Calculate produces the next EMA result given the next input.
func (ema *EMA) Update(next int64) (result float64) {
	ema.Last = next
	if ema.Init {
		ema.Avg = float64(next)*ema.Constant + ema.Avg*(1-ema.Constant)
	} else {
		ema.Init = true
		ema.Avg = float64(next)
	}

	return ema.Avg
}
