// Package labels ...
// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package labels

// Set structure implemented as a map.
type Set map[string]interface{}

// InfoSource represents a prometheus info metric, those are pseudo-metrics
// that provide metadata in the form of labels.
type InfoSource struct {
	Name   string
	Labels Set
}

// DifferenceEqualValues does:
//  - Get all the labels that have are in both A and B label sets
//  - If those labels have the same values in both A and B, returns the difference A - B of label-values and "true"
//  - Otherwise, returns nil and false
//  - If there is no intersection in the label names, returns A and true
func DifferenceEqualValues(a, b Set) (Set, bool) {
	difference := make(Set, len(a))
	for k, v := range a {
		difference[k] = v
	}

	for key, vb := range b {
		if va, ok := a[key]; ok {
			if vb == va {
				delete(difference, key)
			} else {
				return nil, false
			}
		}
	}
	return difference, true
}

// Join returns the labels from src that should be added to dst if the label names in criteria coincide.
// If criteria is empty, returns src
// The function ignores the values in criteria
func Join(src, dst, criteria Set) (Set, bool) {
	ret := Set{}
	for k, v := range src {
		ret[k] = v
	}
	for name := range criteria {
		vs, ok := src[name]
		if !ok {
			return nil, false
		}
		vd, ok := dst[name]
		if !ok {
			return nil, false
		}
		if vs != vd {
			return nil, false
		}
		delete(ret, name)
	}
	return ret, true
}

// ToAdd decide which labels should be added, a set of _info metrics, to the destination label
// set.
// It does, for each info:
// - if DifferenceEqualValues(info, b) == x, true:
//      - suffixes info.Name to all x label names and adds it to the result
// - If info1.Name == info2.Name AND DifferenceEqualValues(info1, b) == x, true and DifferenceEqualValues(info1, b) == y, true:
//      - no metrics neither from info1.Name nor info2.Name are added to the result
func ToAdd(infos []InfoSource, dst Set) Set {
	// Time complexity of this implementation (assuming no hash collisions): O(IxL), where:
	// - I is the number of _info fields
	// - L is the average number of labels that should be added, from each info field

	// key: source info metric, value: labels to be added for this info
	labels := make(map[string]Set, len(infos))
	// info sources that must be ignored because there would be conflicts (same label names, different values)
	ignoredInfos := map[string]interface{}{}

iterateInfos:
	for _, i := range infos {
		if _, ok := ignoredInfos[i.Name]; ok {
			continue
		}
		toAdd, ok := DifferenceEqualValues(i.Labels, dst)
		if !ok {
			continue
		}
		for k, v := range toAdd {
			infoLabels, ok := labels[i.Name]
			if !ok {
				infoLabels = Set{}
				labels[i.Name] = infoLabels
			}
			if alreadyVal, ok := infoLabels[k]; ok && v != alreadyVal {
				// two infos have different coinciding attributes. Discarding this info name
				ignoredInfos[i.Name] = true
				continue iterateInfos
			}
			infoLabels[k] = v
		}
	}

	// Removed ignored _info fields from the initial tree of labels
	for k := range ignoredInfos {
		delete(labels, k)
	}

	// consolidate the tree of labels into a flat map, where each entry is:
	// info_name.label_name = label_value
	flatLabels := Set{}
	for infoName, infoLabels := range labels {
		for k, v := range infoLabels {
			flatLabels[k+"."+infoName] = v
		}
	}
	return flatLabels
}

// Accumulate copies the labels of the source label Set into the destination.
func Accumulate(dst, src Set) {
	for k, v := range src {
		if _, ok := dst[k]; !ok {
			dst[k] = v
		}
	}
}

// AccumulateOnly copies the labels from the source set into the destination, but only those that are present
// in the attrs set
func AccumulateOnly(dst, src, attrs Set) {
	for k := range attrs {
		if v, ok := src[k]; ok {
			if _, ok := dst[k]; !ok {
				dst[k] = v
			}
		}
	}
}
