// Copyright 2015 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package histogram

import (
	"errors"
	"fmt"
	"math"
	"sort"
)

// Percentile calculates the percentile `p` based on the buckets. The
// buckets will be sorted by this function (i.e. no sorting needed before
// calling this function). The percentile value is interpolated assuming a
// linear distribution within a bucket. However, if the percentile falls
// into the highest bucket, the upper bound of the 2nd highest bucket is
// returned. A natural lower bound of 0 is assumed if the upper bound of the
// lowest bucket is greater 0. In that case, interpolation in the lowest
// bucket happens linearly between 0 and the upper bound of the lowest
// bucket. However, if the lowest bucket has an upper bound less or equal to
// 0, this upper bound is returned if the percentile falls into the lowest
// bucket.
//
// An error is returned if:
//  * `buckets` has fewer than 2 elements
//  * the highest bucket is not +Inf
//  * p<0
//  * p>100
func Percentile(p float64, buckets Buckets) (float64, error) {
	if p < 0.0 {
		return 0, fmt.Errorf("invalid percentile: %g (must be greater than 0.0)", p)
	}
	if p > 100.0 {
		return 0, fmt.Errorf("invalid percentile: %g (must be less than 100.0)", p)
	}
	if len(buckets) < 2 {
		return 0, fmt.Errorf("invalid buckets: minimum of 2 buckets required, got %d", len(buckets))
	}
	sort.Sort(buckets)
	if !math.IsInf(buckets[len(buckets)-1].UpperBound, +1) {
		return 0, errors.New("invalid buckets: highest bucket is not +Inf")
	}

	buckets = coalesceBuckets(buckets)
	ensureMonotonic(buckets)

	rank := (p / 100.0) * buckets[len(buckets)-1].Count
	b := sort.Search(len(buckets)-1, func(i int) bool { return buckets[i].Count >= rank })

	if b == len(buckets)-1 {
		return buckets[len(buckets)-2].UpperBound, nil
	}
	if b == 0 && buckets[0].UpperBound <= 0 {
		return buckets[0].UpperBound, nil
	}
	var (
		bucketStart float64
		bucketEnd   = buckets[b].UpperBound
		count       = buckets[b].Count
	)
	if b > 0 {
		bucketStart = buckets[b-1].UpperBound
		count -= buckets[b-1].Count
		rank -= buckets[b-1].Count
	}
	return bucketStart + (bucketEnd-bucketStart)*(rank/count), nil
}

// coalesceBuckets merges buckets with the same upper bound.
//
// The input buckets must be sorted.
func coalesceBuckets(buckets Buckets) Buckets {
	last := buckets[0]
	i := 0
	for _, b := range buckets[1:] {
		if b.UpperBound == last.UpperBound {
			last.Count += b.Count
		} else {
			buckets[i] = last
			last = b
			i++
		}
	}
	buckets[i] = last
	return buckets[:i+1]
}

// The assumption that bucket counts increase monotonically with increasing
// UpperBound may be violated during:
//
//   * Recording rule evaluation of histogram_quantile, especially when rate()
//      has been applied to the underlying bucket timeseries.
//   * Evaluation of histogram_quantile computed over federated bucket
//      timeseries, especially when rate() has been applied.
//
// This is because scraped data is not made available to rule evaluation or
// federation atomically, so some buckets are computed with data from the
// most recent scrapes, but the other buckets are missing data from the most
// recent scrape.
//
// Monotonicity is usually guaranteed because if a bucket with upper bound
// u1 has count c1, then any bucket with a higher upper bound u > u1 must
// have counted all c1 observations and perhaps more, so that c  >= c1.
//
// As a somewhat hacky solution until ingestion is atomic per scrape, we
// calculate the "envelope" of the histogram buckets, essentially removing
// any decreases in the count between successive buckets.
func ensureMonotonic(buckets Buckets) {
	max := buckets[0].Count
	for i := range buckets[1:] {
		switch {
		case buckets[i].Count > max:
			max = buckets[i].Count
		case buckets[i].Count < max:
			buckets[i].Count = max
		}
	}
}
