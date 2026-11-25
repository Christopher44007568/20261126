package main

import (
	"math"
	"math/rand"
)

type rng struct {
	r *rand.Rand
}

func newRNG(seed int64) *rng {
	return &rng{r: rand.New(rand.NewSource(seed))}
}

func (g *rng) gaussian(mean, sigma float64) float64 {
	if sigma <= 0 {
		return mean
	}
	return mean + g.r.NormFloat64()*sigma
}

// poisson returns integer electron counts. The normal approximation is only
// used where lambda is large enough for the central-limit approximation to be
// visually and numerically close to the exact distribution.
func (g *rng) poisson(lambda float64) float64 {
	if lambda <= 0 {
		return 0
	}
	if lambda >= 32 {
		value := math.Round(g.gaussian(lambda, math.Sqrt(lambda)))
		if value < 0 {
			return 0
		}
		return value
	}

	product := 1.0
	limit := math.Exp(-lambda)
	count := 0.0
	for {
		product *= g.r.Float64()
		if product <= limit {
			return count
		}
		count++
	}
}

func clampFloat(value, low, high float64) float64 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func smoothStep(edge0, edge1, x float64) float64 {
	t := clampFloat((x-edge0)/(edge1-edge0), 0, 1)
	return t * t * (3 - 2*t)
}
