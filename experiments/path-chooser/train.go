package main

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"

	decision "github.com/kimjooyoon/gooo-decision-runtime"
)

const width, hidden, labels = decision.FeatureDim, decision.HiddenDim, decision.LabelCount
const w1Count = width * hidden
const w2Start = w1Count + hidden
const b2Start = w2Start + hidden*labels
const parameterCount = b2Start + labels

type network [parameterCount]float32
type example struct {
	X        [width]float32
	Active   []int
	Label    int
	Text, ID string
}
type lossPoint struct {
	Epoch int
	Mean  float64
}

func initialize(seed uint64) network {
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	var n network
	for i := range w1Count {
		n[i] = float32(rng.NormFloat64() * math.Sqrt(2./width))
	}
	for i := w2Start; i < b2Start; i++ {
		n[i] = float32(rng.NormFloat64() * math.Sqrt(1./hidden))
	}
	return n
}

func ternary(values []float32) ([]int8, float32) {
	sum := 0.
	for _, v := range values {
		sum += math.Abs(float64(v))
	}
	scale := float32(max(sum/float64(len(values)), 1e-8))
	codes := make([]int8, len(values))
	for i, v := range values {
		codes[i] = int8(max(-1., min(1., math.Round(float64(v/scale)))))
	}
	return codes, scale
}

func projected(n network, qat bool) network {
	if qat {
		for _, rangeValue := range [][2]int{{0, w1Count}, {w2Start, b2Start}} {
			codes, scale := ternary(n[rangeValue[0]:rangeValue[1]])
			for i, v := range codes {
				n[rangeValue[0]+i] = float32(v) * scale
			}
		}
	}
	return n
}

func logits(n *network, e *example) ([hidden]float32, [labels]float32) {
	var h [hidden]float32
	var z [labels]float32
	for row := range hidden {
		v := n[w1Count+row]
		for _, i := range e.Active {
			v += n[row*width+i] * e.X[i]
		}
		h[row] = max(v, 0)
	}
	for label := range labels {
		z[label] = n[b2Start+label]
		for i, v := range h {
			z[label] += n[w2Start+label*hidden+i] * v
		}
	}
	return h, z
}

func gradient(n *network, e *example, g *[parameterCount]float64) float64 {
	h, z := logits(n, e)
	maximum := float64(z[0])
	for _, v := range z {
		maximum = max(maximum, float64(v))
	}
	var p [labels]float64
	sum := 0.
	for i, v := range z {
		p[i] = math.Exp(float64(v) - maximum)
		sum += p[i]
	}
	loss := -float64(z[e.Label]) + maximum + math.Log(sum)
	for label := range labels {
		d := p[label] / sum
		if label == e.Label {
			d--
		}
		g[b2Start+label] += d
		for i, v := range h {
			g[w2Start+label*hidden+i] += d * float64(v)
			if v <= 0 {
				continue
			}
			dh := d * float64(n[w2Start+label*hidden+i])
			g[w1Count+i] += dh
			for _, x := range e.Active {
				g[i*width+x] += dh * float64(e.X[x])
			}
		}
	}
	return loss
}

func train(ctx context.Context, n network, data []example, epochs int, rate float64, qat bool) (network, []lossPoint, error) {
	if len(data) == 0 || epochs < 1 || epochs > 500 {
		return n, nil, fmt.Errorf("bounded training required")
	}
	var first, second [parameterCount]float64
	order := make([]int, len(data))
	for i := range order {
		order[i] = i
	}
	rng := rand.New(rand.NewPCG(20261010, 991))
	step := 0
	trace := []lossPoint{}
	for epoch := 1; epoch <= epochs; epoch++ {
		if err := ctx.Err(); err != nil {
			return n, trace, err
		}
		rng.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
		loss := 0.
		for start := 0; start < len(order); start += 8 {
			stop := min(start+8, len(order))
			forward := projected(n, qat)
			var g [parameterCount]float64
			for _, i := range order[start:stop] {
				loss += gradient(&forward, &data[i], &g)
			}
			step++
			c1, c2 := 1-math.Pow(.9, float64(step)), 1-math.Pow(.999, float64(step))
			for i := range n {
				v := g[i] / float64(stop-start)
				first[i] = .9*first[i] + .1*v
				second[i] = .999*second[i] + .001*v*v
				n[i] -= float32(rate * (first[i] / c1) / (math.Sqrt(second[i]/c2) + 1e-8))
				if math.IsNaN(float64(n[i])) || math.IsInf(float64(n[i]), 0) {
					return n, trace, fmt.Errorf("nonfinite training")
				}
			}
		}
		trace = append(trace, lossPoint{epoch, loss / float64(len(data))})
	}
	return n, trace, nil
}
