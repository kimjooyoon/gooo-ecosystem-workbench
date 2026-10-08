package main

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
)

const width, hidden = 256, 8
const w1Count = width * hidden
const w2Start = w1Count + hidden
const parameterCount = w2Start + 2*hidden

type network [parameterCount]float32
type fieldExample struct {
	X      [width]float32
	Active []uint16
	Label  int
}
type epochLoss struct {
	Epoch int     `json:"epoch"`
	Mean  float64 `json:"mean_field_cross_entropy"`
}

func examples(rows []row, indices []int) []fieldExample {
	result := make([]fieldExample, 0, len(indices)*3)
	for _, i := range indices {
		for bit := range 3 {
			e := fieldExample{Label: int(rows[i].Label>>bit) & 1}
			copy(e.X[:], rows[i].Full[bit*width:(bit+1)*width])
			for j, v := range e.X {
				if v != 0 {
					e.Active = append(e.Active, uint16(j))
				}
			}
			result = append(result, e)
		}
	}
	return result
}

func initialize(seed uint64) network {
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	var n network
	for i := range w1Count {
		n[i] = float32(rng.NormFloat64() * math.Sqrt(2./width))
	}
	for i := w2Start; i < parameterCount; i++ {
		n[i] = float32(rng.NormFloat64() * math.Sqrt(1./hidden))
	}
	return n
}

func quantize(values []float32) ([]int8, float32) {
	var sum float64
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

type forwardModel struct {
	Values         network
	Codes1, Codes2 []int8
	Scale1, Scale2 float32
}

func forwardWeights(n network, qat bool) forwardModel {
	f := forwardModel{Values: n}
	if qat {
		f.Codes1, f.Scale1 = quantize(n[:w1Count])
		f.Codes2, f.Scale2 = quantize(n[w2Start:])
	}
	return f
}

func (f *forwardModel) forward(e *fieldExample) (h [hidden]float32, logits [2]float32) {
	for row := range hidden {
		var sum float32
		for _, column := range e.Active {
			w := f.Values[row*width+int(column)]
			if f.Codes1 != nil {
				w = float32(f.Codes1[row*width+int(column)])
			}
			sum = float32(sum + float32(e.X[column]*w))
		}
		if f.Codes1 != nil {
			sum = float32(sum * f.Scale1)
		}
		h[row] = max(float32(sum+f.Values[w1Count+row]), 0)
	}
	for label := range 2 {
		for j, v := range h {
			w := f.Values[w2Start+label*hidden+j]
			if f.Codes2 != nil {
				w = float32(f.Codes2[label*hidden+j])
			}
			logits[label] = float32(logits[label] + float32(v*w))
		}
		if f.Codes2 != nil {
			logits[label] = float32(logits[label] * f.Scale2)
		}
	}
	return h, logits
}

func gradient(f *forwardModel, e *fieldExample, temperature float64, g *[parameterCount]float64) float64 {
	h, z := f.forward(e)
	difference := (float64(z[1]) - float64(z[0])) / temperature
	p := 1 / (1 + math.Exp(-difference))
	loss := math.Max(difference, 0) - float64(e.Label)*difference + math.Log1p(math.Exp(-math.Abs(difference)))
	d := (p - float64(e.Label)) / temperature
	for j, v := range h {
		g[w2Start+j] -= d * float64(v)
		g[w2Start+hidden+j] += d * float64(v)
		if v <= 0 {
			continue
		}
		w0, w1 := f.Values[w2Start+j], f.Values[w2Start+hidden+j]
		if f.Codes2 != nil {
			w0 = float32(f.Codes2[j]) * f.Scale2
			w1 = float32(f.Codes2[hidden+j]) * f.Scale2
		}
		dh := d * float64(w1-w0)
		g[w1Count+j] += dh
		for _, column := range e.Active {
			g[j*width+int(column)] += dh * float64(e.X[column])
		}
	}
	return loss
}

func train(ctx context.Context, n network, data []fieldExample, epochs, batch int, rate, temperature float64, seed uint64, qat bool) (network, []epochLoss, error) {
	if len(data) == 0 || batch < 1 || epochs < 1 {
		return n, nil, fmt.Errorf("nonempty bounded training required")
	}
	order := make([]int, len(data))
	for i := range order {
		order[i] = i
	}
	rng := rand.New(rand.NewPCG(seed, seed^0xd1b54a32d192ed03))
	var first, second [parameterCount]float64
	var trace []epochLoss
	step := 0
	for epoch := 1; epoch <= epochs; epoch++ {
		if err := ctx.Err(); err != nil {
			return n, trace, err
		}
		rng.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
		loss := 0.
		for start := 0; start < len(order); start += batch {
			stop := min(start+batch, len(order))
			f := forwardWeights(n, qat)
			var g [parameterCount]float64
			for _, i := range order[start:stop] {
				loss += gradient(&f, &data[i], temperature, &g)
			}
			step++
			correction1, correction2 := 1-math.Pow(.9, float64(step)), 1-math.Pow(.999, float64(step))
			for i := range n {
				v := g[i] / float64(stop-start)
				first[i] = .9*first[i] + .1*v
				second[i] = .999*second[i] + .001*v*v
				n[i] -= float32(rate * (first[i] / correction1) / (math.Sqrt(second[i]/correction2) + 1e-8))
				if math.IsNaN(float64(n[i])) || math.IsInf(float64(n[i]), 0) {
					return n, trace, fmt.Errorf("nonfinite training parameter")
				}
			}
		}
		trace = append(trace, epochLoss{epoch, loss / float64(len(data))})
	}
	return n, trace, nil
}
