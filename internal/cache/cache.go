package cache

import (
	"container/heap"
	"sync"
	"time"
)

type Record struct {
	Key       string
	Value     interface{}
	Priority  int
	Frequency int
	TTL       time.Time
	Index     int // Index in the priority queue
}

type Cache struct {
	Capacity int
	Record   map[string]*Record
	Heap     *PriorityQueue
	Mutex    sync.Mutex
}

type PriorityQueue []*Record

func NewCache(capacity int) *Cache {
	pq := &PriorityQueue{}
	heap.Init(pq)
	return &Cache{
		Capacity: capacity,
		Record:   make(map[string]*Record),
		Heap:     pq,
	}
}

func (c *Cache) Get(key string) (interface{}, bool) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	record, exists := c.Record[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(record.TTL) {
		c.evict(key)
		return nil, false
	}

	record.Frequency++
	heap.Fix(c.Heap, record.Index)
	return record.Value, true
}

func (c *Cache) Set(key string, value interface{}, priority int, ttl time.Duration) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	if record, exists := c.Record[key]; exists {
		// Update existing record.
		record.Value = value
		record.Priority = priority
		record.TTL = time.Now().Add(ttl)
		record.Frequency++
		heap.Fix(c.Heap, record.Index)
	} else {
		// Add new record.
		record = &Record{
			Key:       key,
			Value:     value,
			Priority:  priority,
			Frequency: 1,
			TTL:       time.Now().Add(ttl),
		}
		c.evictIfNeeded()
		heap.Push(c.Heap, record)
		c.Record[key] = record
	}
}

func (c *Cache) Evict(key string) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	record := c.Record[key]
	heap.Remove(c.Heap, record.Index)
	delete(c.Record, key)
}

func (c *Cache) evictIfNeeded() {
	for len(c.Record) >= c.Capacity {
		record := heap.Pop(c.Heap).(*Record)
		delete(c.Record, record.Key)
	}
}

func (c *Cache) evict(key string) {
	record := c.Record[key]
	heap.Remove(c.Heap, record.Index)
	delete(c.Record, key)
}

func (pq *PriorityQueue) Len() int { return len(*pq) }

// Less compare whether the element with index i must sort before the element with index j by checking the following in order
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
	record := x.(*Record)
	record.Index = n
	*pq = append(*pq, record)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	record := old[n-1]
	record.Index = -1 // for safety
	*pq = old[0 : n-1]
	return record
}
