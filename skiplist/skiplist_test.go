package skiplist

import (
	"strconv"
	"testing"
)

func TestSkipList(t *testing.T) {
	sl := New()
	err := sl.Insert(5, "test5")
	if err != nil {
		t.Error(err)
	}

	err = sl.Insert(5, "testxx")
	if err != ErrNodeExists {
		t.Error("node must exists")
	}

	err = sl.Insert(3, "test3")
	if err != nil {
		t.Error(err)
	}

	err = sl.Insert(7, "test7")
	if err != nil {
		t.Error(err)
	}

	err = sl.Insert(9, "test9")
	if err != nil {
		t.Error(err)
	}

	err = sl.Insert(1, "test1")
	if err != nil {
		t.Error(err)
	}

	if sl.Length != 5 {
		t.Error("skiplist Length init 3")
	}

	i, j := 1, int64(1)
	for n := sl.Head.Next(); n != nil; n = n.Next() {
		if n.Member != "test"+strconv.FormatInt(n.Score, 10) {
			t.Errorf("skiplist node member error (%s)", n.Member)
		}

		if n.Score != j {
			t.Error("skiplist node score error")
		}

		i++
		j += 2
	}

	sl.Update(1, "test111")
	if sl.Length != 5 {
		t.Error("skiplist Length init 3")
	}

	n := sl.Equal(1)
	if n == nil {
		t.Error("skiplist update error")
	}

	if n.Member != "test111" {
		t.Error("skiplist update error")
	}

	n = sl.Delete(10)
	if n != nil {
		t.Error("skiplist node delete error")
	}

	n = sl.Delete(9)
	if n == nil {
		t.Error("skiplist node delete error")
	}

	if val, ok := n.Member.(string); !ok {
		t.Error("skiplist node delete error")
	} else {
		if val != "test9" {
			t.Error("skiplist node delete error")
		}
	}

	if sl.Length != 4 {
		t.Error("skiplist Length init 3")
	}

	n = sl.Greate(11)
	if n != nil {
		t.Error("skiplist greate error")
	}

	n = sl.Greate(3)
	if n == nil {
		t.Error("skiplist greate error")
	} else {
		if val, ok := n.Member.(string); !ok {
			t.Error("skiplist greate error")
		} else {
			if val != "test5" {
				t.Error("skiplist greate error")
			}
		}
	}
}

func BenchmarkSkipListInsert(b *testing.B) {
	sl := New()
	for i := 0; i < b.N; i++ {
		err := sl.Insert(int64(i), "test")
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkSkipListEqual(b *testing.B) {
	b.StopTimer()
	sl := New()
	for i := 0; i < b.N; i++ {
		err := sl.Insert(int64(i), "test")
		if err != nil {
			b.Error(err)
		}
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n := sl.Equal(int64(i)); n == nil {
			b.Error("skiplist not find node")
		}
	}
}
