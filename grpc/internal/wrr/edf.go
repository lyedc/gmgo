/*
 *
 * Copyright 2019 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package wrr

import (
	"container/heap"
	"sync"
)

// edfWrr is a struct for EDF weighted round robin implementation.
type edfWrr struct {
	lock               sync.Mutex
	items              edfPriorityQueue
	currentOrderOffset uint64
	currentTime        float64
}

// NewEDF creates Earliest Deadline First (EDF)
// (https://en.wikipedia.org/wiki/Earliest_deadline_first_scheduling) implementation for weighted round robin.
// Each pick from the schedule has the earliest deadline entry selected. Entries have deadlines set
// at current time + 1 / weight, providing weighted round robin behavior with O(log n) pick time.
func NewEDF() WRR {
	return &edfWrr{}
}

// edfEntry is an internal wrapper for item that also stores weight and relative position in the queue.
type edfEntry struct {
	deadline    float64
	weight      int64
	orderOffset uint64
	item        interface{}
}

// edfPriorityQueue is a heap.Interface implementation for edfEntry elements.
type edfPriorityQueue []*edfEntry

func (pq *edfPriorityQueue) Len() int { return len(*pq) }
func (pq *edfPriorityQueue) Less(i, j int) bool {
	edf := *pq
	return edf[i].deadline < edf[j].deadline || edf[i].deadline == edf[j].deadline && edf[i].orderOffset < edf[j].orderOffset
}
func (pq *edfPriorityQueue) Swap(i, j int) {
	edf := *pq
	edf[i], edf[j] = edf[j], edf[i]
}

func (pq *edfPriorityQueue) Push(x interface{}) {
	*pq = append(*pq, x.(*edfEntry))
}

func (pq *edfPriorityQueue) Pop() interface{} {
	old := *pq
	*pq = old[0 : len(old)-1]
	return old[len(old)-1]
}

func (edf *edfWrr) Add(item interface{}, weight int64) {
	edf.lock.Lock()
	defer edf.lock.Unlock()
	entry := edfEntry{
		deadline:    edf.currentTime + 1.0/float64(weight),
		weight:      weight,
		item:        item,
		orderOffset: edf.currentOrderOffset,
	}
	edf.currentOrderOffset++
	heap.Push(&edf.items, &entry)
}

func (edf *edfWrr) Next() interface{} {
	edf.lock.Lock()
	defer edf.lock.Unlock()
	if len(edf.items) == 0 {
		return nil
	}
	item := edf.items[0]
	edf.currentTime = item.deadline
	item.deadline = edf.currentTime + 1.0/float64(item.weight)
	heap.Fix(&edf.items, 0)
	return item.item
}
