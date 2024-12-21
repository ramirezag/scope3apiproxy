package cache

import (
	"container/heap"
	"sync"
	"time"
)

type Item struct {
	Key       string
	Value     interface{}
	Priority  int
	Frequency int
	TTL       time.Time
	Index     int // Index in the priority queue
}

type Cache struct {
	Capacity int
	Items    map[string]*Item
	Heap     *PriorityQueue
	Mutex    sync.Mutex
}

type PriorityQueue []*Item

func NewCache(capacity int) *Cache {
	pq := &PriorityQueue{}
	heap.Init(pq)
	return &Cache{
		Capacity: capacity,
		Items:    make(map[string]*Item),
		Heap:     pq,
	}
}

func (c *Cache) Get(key string) (interface{}, bool) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	item, exists := c.Items[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(item.TTL) {
		c.evict(key)
		return nil, false
	}

	item.Frequency++
	heap.Fix(c.Heap, item.Index)
	return item.Value, true
}

func (c *Cache) Set(key string, value interface{}, priority int, ttl time.Duration) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	if item, exists := c.Items[key]; exists {
		// Update existing item.
		item.Value = value
		item.Priority = priority
		item.TTL = time.Now().Add(ttl)
		item.Frequency++
		heap.Fix(c.Heap, item.Index)
	} else {
		// Add new item.
		item = &Item{
			Key:       key,
			Value:     value,
			Priority:  priority,
			Frequency: 1,
			TTL:       time.Now().Add(ttl),
		}
		c.evictIfNeeded()
		heap.Push(c.Heap, item)
		c.Items[key] = item
	}
}

func (c *Cache) evictIfNeeded() {
	for len(c.Items) > c.Capacity {
		item := heap.Pop(c.Heap).(*Item)
		delete(c.Items, item.Key)
	}
}

func (c *Cache) evict(key string) {
	item := c.Items[key]
	heap.Remove(c.Heap, item.Index)
	delete(c.Items, key)
}

func (pq *PriorityQueue) Len() int { return len(*pq) }

// Less compare cache item at index i against item at index j by checking the following in order
// 1. Priority
// 2. Frequency (Least Frequently Used)
// 3. TTL
// returns true if any of the conditions are true. Otherwise, false
func (pq *PriorityQueue) Less(i, j int) bool {
	q := *pq
	if q[i].Priority == q[j].Priority {
		if q[i].Frequency == q[j].Frequency {
			return q[i].TTL.Before(q[j].TTL)
		}
		return q[i].Frequency < q[j].Frequency
	}
	return q[i].Priority < q[j].Priority
}

func (pq *PriorityQueue) Swap(i, j int) {
	q := *pq
	q[i], q[j] = q[j], q[i]
	q[i].Index = i
	q[j].Index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*Item)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.Index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}
