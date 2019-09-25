package histogram

import (
	"math"
	"testing"
)

func TestEnsureMonotonic(t *testing.T) {
	buckets := Buckets{
		{0.5, 2.0},
		{1.0, 1.0},
		{2.0, 3.0},
	}

	ensureMonotonic(buckets)

	expectedCounts := []float64{2.0, 2.0, 3.0}
	for i, want := range expectedCounts {
		if got := buckets[i].Count; got != want {
			t.Errorf("ensureMonotonic failed to ensure monotonicity: buckets[%d] = %f; want %f", i, got, want)
		}
	}

	// Run again to ensure nothing changes for a valid Buckets.
	ensureMonotonic(buckets)
	for i, want := range expectedCounts {
		if got := buckets[i].Count; got != want {
			t.Errorf("ensureMonotonic modified valid monotonicity: buckets[%d] = %f; want %f", i, got, want)
		}
	}
}

func TestCoalesceBuckets(t *testing.T) {
	tests := []struct {
		input Buckets
		want  Buckets
	}{
		{
			Buckets{{0.5, 1.0}, {0.5, 2.0}},
			Buckets{{0.5, 3.0}},
		},
		{
			Buckets{{0.5, 1.0}, {0.5, 2.0}, {1.0, 3.0}},
			Buckets{{0.5, 3.0}, {1.0, 3.0}},
		},
		{
			Buckets{{0.1, 1.0}, {0.5, 1.0}, {0.5, 2.0}, {1.0, 3.0}},
			Buckets{{0.1, 1.0}, {0.5, 3.0}, {1.0, 3.0}},
		},
	}

	for _, test := range tests {
		coalesced := coalesceBuckets(test.input)
		if len(coalesced) != len(test.want) {
			t.Errorf("coalesceBuckets failed to coalesce to desired length %d; got %d", len(test.want), len(coalesced))
			continue
		}
		for i, got := range coalesced {
			want := test.want[i]
			if got.UpperBound != want.UpperBound || got.Count != want.Count {
				t.Errorf("coalesceBuckets failed to coalesce element %d: got %#v; want %#v", i, got, want)
			}
		}
	}
}

func TestPercentileValidatesInput(t *testing.T) {
	tests := []struct {
		p       float64
		buckets Buckets
	}{
		// p < 0.0
		{-1.0, Buckets{}},
		// p > 100.0
		{101.0, Buckets{}},
		// len(buckets) < 2
		{50.0, Buckets{}},
		// len(buckets) < 2
		{50.0, Buckets{{math.Inf(1), 1.0}}},
		// Last bucket is not +Inf
		{50.0, Buckets{{0.5, 1.0}, {1.0, 2.0}}},
	}

	for _, test := range tests {
		if _, err := Percentile(test.p, test.buckets); err == nil {
			t.Errorf("Percentile(%g, %#v) did not return an error", test.p, test.buckets)
		}
	}
}

func TestPercentile(t *testing.T) {
	buckets := Buckets{
		{-20.0, 10.0},
		{1.0, 11.0},
		{40.0, 20.0},
		{200.0, 200.0},
		{math.Inf(1), 220.0},
	}

	tests := []struct {
		p    float64
		want float64
	}{
		{0.0, -20.0},
		{4.0, -20.0},
		{5.0, 1.0},
		{50.0, 120},
		{99.0, 200.0},
		{100.0, 200.0},
	}

	for _, test := range tests {
		if got, _ := Percentile(test.p, buckets); got != test.want {
			t.Errorf("Percentile(%g, %v) = %g; want %g", test.p, buckets, got, test.want)
		}
	}
}

var benchmarkBuckets = Buckets{
	{10.0, 10.0},
	{20.0, 20.0},
	{30.0, 30.0},
	{math.Inf(1), 30.0},
}

func BenchmarkPercentile(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = Percentile(0.5, benchmarkBuckets)
	}
}
