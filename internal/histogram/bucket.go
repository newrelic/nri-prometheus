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

// Bucket represents a single grouping of values from Prometheus histogram metrics.
type Bucket struct {
	UpperBound float64
	Count      float64
}

// Buckets implements sort.Interface.
type Buckets []Bucket

func (b Buckets) Len() int           { return len(b) }
func (b Buckets) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b Buckets) Less(i, j int) bool { return b[i].UpperBound < b[j].UpperBound }
