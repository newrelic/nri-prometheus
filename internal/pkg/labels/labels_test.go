// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package labels

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDifferenceEqualValues(t *testing.T) {
	cases := []struct {
		a     Set  // source labels
		b     Set  // labels whose intersection with A would be subtracted from A
		exp   Set  // expected result Set
		match bool // if we expect that common a and b labels match
	}{
		{
			a:     Set{"container": "c1", "container_id": "cid1", "image": "i1", "image_id": "iid1", "namespace": "ns1", "pod": "p1"},
			b:     Set{"container": "c1", "namespace": "ns1", "node": "n1", "pod": "p1"},
			exp:   Set{"container_id": "cid1", "image": "i1", "image_id": "iid1"},
			match: true,
		},
		{
			a:     Set{"container": "c1", "container_id": "cid1", "image": "i1", "image_id": "iid1", "namespace": "ns1", "pod": "p1"},
			b:     Set{"container": "different", "namespace": "ns1", "node": "n1", "pod": "p1"},
			exp:   nil,
			match: false,
		},
		{
			a:     Set{"container": "c1", "container_id": "cid1", "image": "i1", "image_id": "iid1", "namespace": "ns1", "pod": "p1"},
			b:     Set{"container": "different", "namespace": "ns1", "node": "n1", "pod": "p1"},
			exp:   nil,
			match: false,
		},
		{
			a:     Set{"a": "b", "c": "d", "e": "f"},
			b:     Set{"g": "h", "i": "j"},
			exp:   Set{"a": "b", "c": "d", "e": "f"},
			match: true,
		},
		{
			a:     Set{},
			b:     Set{"g": "h", "i": "j"},
			exp:   Set{},
			match: true,
		},
		{
			a:     Set{"a": "b", "c": "d", "e": "f"},
			b:     Set{},
			exp:   Set{"a": "b", "c": "d", "e": "f"},
			match: true,
		},
		{
			a:     Set{},
			b:     Set{},
			exp:   Set{},
			match: true,
		},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("match: %v exp: %v", c.match, c.exp), func(t *testing.T) {
			i, ok := DifferenceEqualValues(c.a, c.b)
			assert.Equal(t, c.match, ok)
			assert.Equal(t, c.exp, i)
		})
	}
}

func TestToAdd(t *testing.T) {
	cases := []struct {
		name  string
		infos []InfoSource // Info labels
		dst   Set          // Labes where info labels would be added
		exp   Set          // expected labels that should be added Set
	}{
		{
			name: "same set of coinciding labels",
			infos: []InfoSource{
				{
					Name:   "some_info",
					Labels: Set{"container": "c1", "container_id": "cid1", "image": "i1", "image_id": "iid1", "namespace": "ns1", "pod": "p1"},
				},
				{
					Name:   "other_info",
					Labels: Set{"container": "c1", "namespace": "ns1", "node": "n1", "pod": "p1", "stuff": 356},
				},
			},
			dst: Set{"container": "c1", "namespace": "ns1", "node": "n1", "pod": "p1"},
			exp: Set{"container_id.some_info": "cid1", "image.some_info": "i1", "image_id.some_info": "iid1", "stuff.other_info": 356},
		},
		{
			name: "different set of coinciding labels",
			infos: []InfoSource{
				{
					Name:   "some_info",
					Labels: Set{"container": "c1", "container_id": "cid1", "image": "i1", "image_id": "iid1", "namespace": "ns1", "pod": "p1"},
				},
				{
					Name:   "other_info",
					Labels: Set{"container": "c1", "node": "n1", "pod": "p1", "stuff": 356}, // namespace does not coincide
				},
			},
			dst: Set{"container": "c1", "namespace": "ns1", "node": "n1", "pod": "p1"},
			exp: Set{"container_id.some_info": "cid1", "image.some_info": "i1", "image_id.some_info": "iid1", "stuff.other_info": 356},
		},
		{
			name: "other_info does not coincide in a label value",
			infos: []InfoSource{
				{
					Name:   "some_info",
					Labels: Set{"container": "c1", "container_id": "cid1", "image": "i1", "image_id": "iid1", "namespace": "ns1", "pod": "p1"},
				},
				{
					Name:   "other_info",
					Labels: Set{"container": "c1bis", "node": "n1", "pod": "p1", "stuff": 356},
				},
			},
			dst: Set{"container": "c1", "namespace": "ns1", "node": "n1", "pod": "p1"},
			exp: Set{"container_id.some_info": "cid1", "image.some_info": "i1", "image_id.some_info": "iid1"}, // other_info Labels are not added
		},
		{
			name: "no label coincidence in destination label set",
			infos: []InfoSource{
				{
					Name:   "some_info",
					Labels: Set{"container": "c1", "container_id": "cid1", "image": "i1", "image_id": "iid1", "namespace": "ns1", "pod": "p1"},
				},
				{
					Name:   "other_info",
					Labels: Set{"container": "c1", "node": "n1", "pod": "p1", "stuff": 356},
				},
			},
			dst: Set{"a": "b", "c": "d", "f": "g"},
			// All the labels from info sources are going to be added. Please observe that some labels will be added by duplicate
			exp: Set{
				"container.some_info": "c1", "container_id.some_info": "cid1", "image.some_info": "i1", "image_id.some_info": "iid1", "namespace.some_info": "ns1", "pod.some_info": "p1",
				"container.other_info": "c1", "node.other_info": "n1", "pod.other_info": "p1", "stuff.other_info": 356,
			},
		},
		{
			name: "definitely not belonging to the same entity",
			infos: []InfoSource{
				{
					Name:   "some_info",
					Labels: Set{"container": "c2", "container_id": "cid1", "image": "i1", "image_id": "iid1", "namespace": "ns1", "pod": "p1"},
				},
				{
					Name:   "other_info",
					Labels: Set{"container": "c3", "namespace": "ns1", "node": "n1", "pod": "p1", "stuff": 356},
				},
			},
			dst: Set{"container": "c1", "namespace": "ns1", "node": "n1", "pod": "p1"},
			exp: Set{}, // despite many labels in common, no labels are going to be added since container differs
		},
		{
			name: "infos with the same name and some common labels",
			infos: []InfoSource{
				{
					Name:   "some_info",
					Labels: Set{"container": "c1", "namespace": "ns1", "pod": "p1", "something": "cool"},
				},
				{
					Name:   "some_info",
					Labels: Set{"container": "c1", "namespace": "ns1", "pod": "p1", "stuff": 356},
				},
			},
			dst: Set{"container": "c1", "namespace": "ns1", "pod": "p1"},
			exp: Set{"something.some_info": "cool", "stuff.some_info": 356},
		},
		{
			name: "infos with the same name and a different-value, same-name label",
			infos: []InfoSource{
				{
					Name:   "some_info",
					Labels: Set{"container": "c1", "namespace": "ns1", "pod": "p1", "something": "cool", "discarding_id": "12345"},
				},
				{
					Name:   "some_info",
					Labels: Set{"container": "c1", "namespace": "ns1", "pod": "p1", "stuff": 356, "discarding_id": "12345"},
				},
				{
					Name:   "some_info",
					Labels: Set{"container": "c1", "namespace": "ns1", "pod": "p1", "discarding_id": "33333"},
				},
			},
			dst: Set{"container": "c1", "namespace": "ns1", "pod": "p1"},
			// Since we cannot be sure whether we should apply metrics from discarding_id == 12345 or 333333, we don't add any of them
			exp: Set{},
		},
		{
			name: "infos with the same name and a different-value, same-name label. Other infos can be added",
			infos: []InfoSource{
				{
					Name:   "some_info",
					Labels: Set{"container": "c1", "namespace": "ns1", "pod": "p1", "something": "cool", "discarding_id": "12345"},
				},
				{
					Name:   "cool_stuff",
					Labels: Set{"container": "c1", "tracatra": "tracatra"},
				},
				{
					Name:   "some_info",
					Labels: Set{"container": "c1", "namespace": "ns1", "pod": "p1", "stuff": 356, "discarding_id": "12345"},
				},
				{
					Name:   "some_info",
					Labels: Set{"container": "c1", "namespace": "ns1", "pod": "p1", "discarding_id": "33333"},
				},
			},
			dst: Set{"container": "c1", "namespace": "ns1", "pod": "p1"},
			// Since we cannot be sure whether we should apply metrics from discarding_id == 12345 or 333333, we don't add any of them
			exp: Set{"tracatra.cool_stuff": "tracatra"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			i := ToAdd(c.infos, c.dst)
			assert.Equal(t, c.exp, i)
		})
	}
}

func TestAccumulate(t *testing.T) {
	cases := []struct {
		dst Set
		src Set
		exp Set // expected union
	}{
		{
			dst: Set{"a": "b", "c": "d"},
			src: Set{"e": "f", "g": "h"},
			exp: Set{"a": "b", "c": "d", "e": "f", "g": "h"},
		},
		{
			dst: Set{"a": "b", "c": "d"},
			src: Set{},
			exp: Set{"a": "b", "c": "d"},
		},
		{
			dst: Set{},
			src: Set{"e": "f", "g": "h"},
			exp: Set{"e": "f", "g": "h"},
		},
		{
			dst: Set{"a": "b", "c": "d"},
			src: Set{"e": "f", "a": "c"},
			exp: Set{"a": "b", "c": "d", "e": "f"}, // in case of collision, old labels are kept
		},
	}
	for i, c := range cases {
		t.Run(fmt.Sprint("case", i), func(t *testing.T) {
			Accumulate(c.dst, c.src)
			assert.Equal(t, c.exp, c.dst)
		})
	}
}
