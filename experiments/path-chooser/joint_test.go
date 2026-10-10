package main

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func jointFixture() jointExample {
	a, b := example{Active: []int{0}}, example{Active: []int{1}}
	a.X[0], b.X[1] = 1, 1
	return jointExample{Sites: []example{a, b}, Allowed: [][2]int{{4, 5}, {4, 5}}, Masks: []uint16{1, 2}}
}

func TestFrozenJointCorpusAndOriginalCheckpoint(t *testing.T) {
	e, err := loadJointCorpus("../../" + jointCorpusRoot)
	if err != nil || len(e.Sites) != 3 || len(e.Masks) != 2 || e.Masks[0] != 1 || e.Masks[1] != 2 {
		t.Fatal(e, err)
	}
	n, err := loadOriginalFP32("../../" + originalFP32Root)
	if err != nil {
		t.Fatal(err)
	}
	data, _, err := corpus(".")
	if err != nil {
		t.Fatal(err)
	}
	first, trace, err := trainJoint(context.Background(), n, data, e, 2, .002, false)
	if err != nil || len(trace) != 2 || trace[1].Steps != 12 {
		t.Fatal(trace, err)
	}
	second, _, err := trainJoint(context.Background(), n, data, e, 2, .002, false)
	if err != nil || first != second || first == n {
		t.Fatal("joint update is absent or nondeterministic", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	stopped, _, err := trainJoint(ctx, n, data, e, 2, .002, false)
	if err != context.Canceled || stopped != n {
		t.Fatal("cancelled joint training changed weights", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "targets.json.gz"), []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadJointCorpus(dir + "/"); err == nil {
		t.Fatal("changed finite targets accepted")
	}
}

func TestJointObjectiveDerivativeAndDirectionRelabeling(t *testing.T) {
	n, e := initialize(42), jointFixture()
	for i := range hidden {
		n[w1Count+i] = .4
	}
	var g [parameterCount]float64
	loss, err := jointGradient(&n, e, &g)
	if err != nil || loss < 0 || math.IsNaN(loss) {
		t.Fatal(loss, err)
	}
	for _, at := range []int{0, 1, w1Count, w2Start + 4*hidden, b2Start + 4, b2Start + 5} {
		plus, minus := n, n
		plus[at] += .001
		minus[at] -= .001
		var scratch [parameterCount]float64
		x, err := jointGradient(&plus, e, &scratch)
		if err != nil {
			t.Fatal(err)
		}
		y, err := jointGradient(&minus, e, &scratch)
		if err != nil {
			t.Fatal(err)
		}
		want := (x - y) / .002
		if math.Abs(g[at]-want) > .002 {
			t.Fatal(at, g[at], want)
		}
	}
	e.Allowed[0] = [2]int{5, 4}
	e.Masks = []uint16{0, 3}
	var relabeled [parameterCount]float64
	other, err := jointGradient(&n, e, &relabeled)
	if err != nil || math.Abs(other-loss) > 1e-12 {
		t.Fatal(other, loss, err)
	}
	for i := range g {
		if math.Abs(g[i]-relabeled[i]) > 1e-12 {
			t.Fatal("label ordering changed objective", i)
		}
	}
}

func TestJointTargetSetDiffersFromIndependentMarginals(t *testing.T) {
	n, e := network{}, jointFixture()
	var g [parameterCount]float64
	loss, err := jointGradient(&n, e, &g)
	if err != nil || math.Abs(loss-math.Log(2)) > 1e-12 {
		t.Fatal(loss, err)
	}
	// Both options appear at both sites. Treating them independently admits all
	// four masks and incorrectly declares zero loss at this same network.
	e.Masks = []uint16{0, 1, 2, 3}
	var all [parameterCount]float64
	loss, err = jointGradient(&n, e, &all)
	if err != nil || math.Abs(loss) > 1e-12 {
		t.Fatal(loss, err)
	}
	for _, v := range all {
		if math.Abs(v) > 1e-12 {
			t.Fatal("all paths accepted but nonzero gradient", v)
		}
	}
}

func TestJointTargetsRejectMissingAndRepeatedEvidence(t *testing.T) {
	n, e := initialize(8), jointFixture()
	for _, masks := range [][]uint16{nil, {1, 1}, {4}} {
		e.Masks = masks
		var g [parameterCount]float64
		if _, err := jointGradient(&n, e, &g); err == nil {
			t.Fatal("invalid targets accepted", masks)
		}
	}
}
