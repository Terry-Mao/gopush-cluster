package skiplist

import (
	"errors"
	"math/rand"
)

const (
	MaxNumberOfLevels = 16
	MaxLevel          = MaxNumberOfLevels - 1
	BitsInRandom      = 31
)

var (
	ErrNodeExists    = errors.New("node already exists")
	ErrNodeNotExists = errors.New("node not exists")
)

type Node struct {
	Score   int64       // key
	Member  interface{} // data
	level   int         // node level
	forward []*Node     // forward index
}

type SkipList struct {
	Level  int   // max level
	Length int   // node length
	Head   *Node // head node
}

func randomLevel() int {
	randomBits := rand.Int()
	randomsLeft := BitsInRandom / 2
	level := 0
	b := 0

	for b == 0 {
		b = randomBits & 3
		if b == 0 {
			level = level + 1
		}

		randomBits = randomBits >> 2
		if randomsLeft = randomsLeft - 1; randomsLeft == 0 {
			randomBits = rand.Int()
			randomsLeft = BitsInRandom / 2
		}
	}

	if level > MaxLevel {
		return MaxLevel
	}

	return level
}

func newNode(level int) *Node {
	return &Node{level: level, forward: make([]*Node, level+1)}
}

// create a a new SkipList
func New() *SkipList {
	sl := &SkipList{}
	sl.Level = 0
	sl.Length = 0
	sl.Head = newNode(MaxNumberOfLevels)

	// init the head node and point to the nil
	for i := 0; i < MaxNumberOfLevels; i++ {
		sl.Head.forward[i] = nil
	}

	return sl
}

// search a node which equals score
func (sl *SkipList) Equal(score int64) *Node {
	var q *Node
	p := sl.Head

	// search from the top forward index
	for i := sl.Level; i >= 0; i-- {
		for q = p.forward[i]; q != nil && q.Score < score; q = p.forward[i] {
			// find next node
			p = q
		}
	}

	// till the bottom index edge
	if q == nil || q.Score != score {
		return nil
	}

	return q
}

// search a node which greate score
func (sl *SkipList) Greate(score int64) *Node {
	var q *Node
	p := sl.Head

	// search from the top forward index
	for i := sl.Level; i >= 0; i-- {
		for q = p.forward[i]; q != nil && q.Score <= score; q = p.forward[i] {
			// find next node
			p = q
		}
	}

	// till the bottom index edge
	if q != nil && q.Score > score {
		return q
	}

	return nil
}

// insert a node into skiplist
func (sl *SkipList) Insert(score int64, member interface{}) error {
	var q *Node
	p := sl.Head
	update := make([]*Node, MaxNumberOfLevels)

	// get all level index the max node which less than the val
	for i := sl.Level; i >= 0; i-- {
		for q = p.forward[i]; q != nil && q.Score < score; q = p.forward[i] {
			p = q
		}

		update[i] = p
	}

	// node exists
	if q != nil && q.Score == score {
		// q.Member = member
		return ErrNodeExists
	}

	// get a random level
	level := randomLevel()
	if level > sl.Level {
		sl.Level = sl.Level + 1
		level = sl.Level
		update[level] = sl.Head
	}

	// new node
	q = newNode(level)
	q.Score = score
	q.Member = member

	// every level index add the new node
	for i := level; i >= 0; i-- {
		p = update[i]
		q.forward[i] = p.forward[i]
		p.forward[i] = q
	}

	sl.Length = sl.Length + 1
	return nil
}

// update a node in skiplist
func (sl *SkipList) Update(score int64, member interface{}) {
	var q *Node
	p := sl.Head
	update := make([]*Node, MaxNumberOfLevels)

	// get all level index the max node which less than the val
	for i := sl.Level; i >= 0; i-- {
		for q = p.forward[i]; q != nil && q.Score < score; q = p.forward[i] {
			p = q
		}

		update[i] = p
	}

	// node exists
	if q != nil && q.Score == score {
		q.Member = member
		return
	}

	// get a random level
	level := randomLevel()
	if level > sl.Level {
		sl.Level = sl.Level + 1
		level = sl.Level
		update[level] = sl.Head
	}

	// new node
	q = newNode(level)
	q.Member = member
	q.Score = score

	// every level index add the new node
	for i := level; i >= 0; i-- {
		p = update[i]
		q.forward[i] = p.forward[i]
		p.forward[i] = q
	}

	sl.Length = sl.Length + 1
	return
}

// delete a node search by score
func (sl *SkipList) Delete(score int64) *Node {
	var q *Node
	p := sl.Head
	update := make([]*Node, MaxNumberOfLevels)

	// every index find the first greate score node
	for i := sl.Level; i >= 0; i-- {
		for q = p.forward[i]; q != nil && q.Score < score; q = p.forward[i] {
			p = q
		}

		update[i] = p
	}

	// found the node
	if q != nil && q.Score == score {
		// update every index's forward (the exists node has deleted)
		for i := 0; i <= sl.Level; i++ {
			p = update[i]
			if q == p.forward[i] {
				p.forward[i] = q.forward[i]
			}
		}

		// every index may delete the last node, so recalc the skiplist's level
		j := sl.Level
		for sl.Head.forward[j] == nil && j > 0 {
			j--
		}

		sl.Level = j
		sl.Length = sl.Length - 1
		return q
	}

	return nil
}

// skiplist node's Next node
func (n *Node) Next() *Node {
	if p := n.forward[0]; p != nil {
		return p
	}

	return nil
}
