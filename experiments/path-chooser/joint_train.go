package main

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
)

type jointLossPoint struct {
	Epoch     int     `json:"epoch"`
	Steps     int     `json:"steps"`
	LabelLoss float64 `json:"mean_label_loss"`
	JointLoss float64 `json:"mean_joint_loss"`
}

func trainJoint(ctx context.Context, n network, data []example, target jointExample,
	epochs int, rate float64, qat bool) (network, []jointLossPoint, error) {
	if err := target.validate(); err != nil {
		return n, nil, err
	}
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
	trace := []jointLossPoint{}
	for epoch := 1; epoch <= epochs; epoch++ {
		if err := ctx.Err(); err != nil {
			return n, trace, err
		}
		rng.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
		labelSum, jointSum, batches := 0., 0., 0
		for start := 0; start < len(order); start += 8 {
			stop := min(start+8, len(order))
			forward := projected(n, qat)
			var g, jointG [parameterCount]float64
			for _, i := range order[start:stop] {
				labelSum += gradient(&forward, &data[i], &g)
			}
			loss, err := jointGradient(&forward, target, &jointG)
			if err != nil {
				return n, trace, err
			}
			jointSum += loss
			batches++
			step++
			c1, c2 := 1-math.Pow(.9, float64(step)), 1-math.Pow(.999, float64(step))
			for i := range n {
				v := .5 * (g[i]/float64(stop-start) + jointG[i])
				first[i] = .9*first[i] + .1*v
				second[i] = .999*second[i] + .001*v*v
				n[i] -= float32(rate * (first[i] / c1) / (math.Sqrt(second[i]/c2) + 1e-8))
				if math.IsNaN(float64(n[i])) || math.IsInf(float64(n[i]), 0) {
					return n, trace, fmt.Errorf("nonfinite training")
				}
			}
		}
		trace = append(trace, jointLossPoint{epoch, step, labelSum / float64(len(data)), jointSum / float64(batches)})
	}
	return n, trace, nil
}
