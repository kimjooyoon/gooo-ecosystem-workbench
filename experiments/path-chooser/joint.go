package main

import (
	"fmt"
	"math"
)

// jointExample retains whole compatible selections. Marginal label lists cannot
// substitute for these masks: combining individually valid options can fail.
type jointExample struct {
	Sites   []example
	Allowed [][2]int
	Masks   []uint16
}

func (e jointExample) validate() error {
	if len(e.Sites) == 0 || len(e.Sites) > 6 || len(e.Allowed) != len(e.Sites) || len(e.Masks) == 0 || len(e.Masks) > 1<<len(e.Sites) {
		return fmt.Errorf("bounded sites, declared label pairs and nonempty joint targets required")
	}
	for _, pair := range e.Allowed {
		if pair[0] < 0 || pair[0] >= labels || pair[1] < 0 || pair[1] >= labels || pair[0] == pair[1] {
			return fmt.Errorf("distinct in-vocabulary candidate pair required")
		}
	}
	var seen [64]bool
	for _, mask := range e.Masks {
		if int(mask) >= 1<<len(e.Sites) || seen[mask] {
			return fmt.Errorf("duplicate or out-of-palette compatible mask")
		}
		seen[mask] = true
	}
	return nil
}

// The objective is -log(sum_{compatible masks} product_{sites} P(option|pair)).
// Its soft targets are the posterior site marginals conditioned on the complete
// compatible set; they are recomputed from this model, never fixed as labels.
// The forward distribution still factorizes. It need not represent a correlated
// multimodal distribution; later native evaluation must measure that limit.
func jointGradient(n *network, e jointExample, g *[parameterCount]float64) (float64, error) {
	if err := e.validate(); err != nil {
		return 0, err
	}
	var h [6][hidden]float32
	var logPair [6][2]float64
	for i := range e.Sites {
		var z [labels]float32
		h[i], z = logits(n, &e.Sites[i])
		pair := e.Allowed[i]
		a, b := float64(z[pair[0]]), float64(z[pair[1]])
		m := max(a, b)
		normalizer := m + math.Log(math.Exp(a-m)+math.Exp(b-m))
		logPair[i] = [2]float64{a - normalizer, b - normalizer}
	}
	var mass [64]float64
	maximum := math.Inf(-1)
	for j, mask := range e.Masks {
		for i := range e.Sites {
			mass[j] += logPair[i][(mask>>i)&1]
		}
		maximum = max(maximum, mass[j])
	}
	total := 0.
	for j := range e.Masks {
		total += math.Exp(mass[j] - maximum)
	}
	logAccepted := maximum + math.Log(total)
	var target [6][2]float64
	for j, mask := range e.Masks {
		posterior := math.Exp(mass[j] - logAccepted)
		for i := range e.Sites {
			target[i][(mask>>i)&1] += posterior
		}
	}
	for i := range e.Sites {
		for option, label := range e.Allowed[i] {
			d := math.Exp(logPair[i][option]) - target[i][option]
			g[b2Start+label] += d
			for j, v := range h[i] {
				g[w2Start+label*hidden+j] += d * float64(v)
				if v <= 0 {
					continue
				}
				dh := d * float64(n[w2Start+label*hidden+j])
				g[w1Count+j] += dh
				for _, x := range e.Sites[i].Active {
					g[j*width+x] += dh * float64(e.Sites[i].X[x])
				}
			}
		}
	}
	return -logAccepted, nil
}
