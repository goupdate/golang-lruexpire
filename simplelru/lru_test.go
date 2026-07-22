package simplelru

import (
	"testing"
	"time"
)

func TestLRU(t *testing.T) {
	evictCounter := 0
	onEvicted := func(k interface{}, v interface{}) {
		if k != v {
			t.Fatalf("Evict values not equal (%v!=%v)", k, v)
		}
		evictCounter += 1
	}
	l, err := NewLRU(128, onEvicted)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	for i := 0; i < 256; i++ {
		l.Add(i, i)
	}
	if l.Len() != 128 {
		t.Fatalf("bad len: %v", l.Len())
	}

	if evictCounter != 128 {
		t.Fatalf("bad evict count: %v", evictCounter)
	}

	for i, k := range l.Keys() {
		if v, ok := l.Get(k); !ok || v != k || v != i+128 {
			t.Fatalf("bad key: %v", k)
		}
	}
	for i := 0; i < 128; i++ {
		_, ok := l.Get(i)
		if ok {
			t.Fatalf("should be evicted")
		}
	}
	for i := 128; i < 256; i++ {
		_, ok := l.Get(i)
		if !ok {
			t.Fatalf("should not be evicted")
		}
	}
	for i := 128; i < 192; i++ {
		ok := l.Remove(i)
		if !ok {
			t.Fatalf("should be contained")
		}
		ok = l.Remove(i)
		if ok {
			t.Fatalf("should not be contained")
		}
		_, ok = l.Get(i)
		if ok {
			t.Fatalf("should be deleted")
		}
	}

	l.Get(192) // expect 192 to be last key in l.Keys()

	for i, k := range l.Keys() {
		if (i < 63 && k != i+193) || (i == 63 && k != 192) {
			t.Fatalf("out of order key: %v", k)
		}
	}

	l.Purge()
	if l.Len() != 0 {
		t.Fatalf("bad len: %v", l.Len())
	}
	if _, ok := l.Get(200); ok {
		t.Fatalf("should contain nothing")
	}
}

func TestLRU_GetOldest_RemoveOldest(t *testing.T) {
	l, err := NewLRU(128, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	for i := 0; i < 256; i++ {
		l.Add(i, i)
	}
	k, _, ok := l.GetOldest()
	if !ok {
		t.Fatalf("missing")
	}
	if k.(int) != 128 {
		t.Fatalf("bad: %v", k)
	}

	k, _, ok = l.RemoveOldest()
	if !ok {
		t.Fatalf("missing")
	}
	if k.(int) != 128 {
		t.Fatalf("bad: %v", k)
	}

	k, _, ok = l.RemoveOldest()
	if !ok {
		t.Fatalf("missing")
	}
	if k.(int) != 129 {
		t.Fatalf("bad: %v", k)
	}
}

// Test that Add returns true/false if an eviction occurred
func TestLRU_Add(t *testing.T) {
	evictCounter := 0
	onEvicted := func(k interface{}, v interface{}) {
		evictCounter += 1
	}

	l, err := NewLRU(1, onEvicted)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if l.Add(1, 1) == true || evictCounter != 0 {
		t.Errorf("should not have an eviction")
	}
	if l.Add(2, 2) == false || evictCounter != 1 {
		t.Errorf("should have an eviction")
	}
}

// Test that Contains doesn't update recent-ness
func TestLRU_Contains(t *testing.T) {
	l, err := NewLRU(2, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	l.Add(1, 1)
	l.Add(2, 2)
	if !l.Contains(1) {
		t.Errorf("1 should be contained")
	}

	l.Add(3, 3)
	if l.Contains(1) {
		t.Errorf("Contains should not have updated recent-ness of 1")
	}
}

// Test that Peek doesn't update recent-ness
func TestLRU_Peek(t *testing.T) {
	l, err := NewLRU(2, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	l.Add(1, 1)
	l.Add(2, 2)
	if v, ok := l.Peek(1); !ok || v != 1 {
		t.Errorf("1 should be set to 1: %v, %v", v, ok)
	}

	l.Add(3, 3)
	if l.Contains(1) {
		t.Errorf("should not have updated recent-ness of 1")
	}
}

// Test that Get removes expired entries from the cache (Len decreases)
func TestLRU_Get_RemovesExpired(t *testing.T) {
	l, err := NewLRUWithExpire(10, 10*time.Millisecond, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	l.Add(1, "value")
	if l.Len() != 1 {
		t.Fatalf("bad len after add: %v", l.Len())
	}

	time.Sleep(20 * time.Millisecond) // let it expire

	v, ok := l.Get(1)
	if ok {
		t.Fatalf("Get should return false for expired entry, got: %v", v)
	}
	if v != nil {
		t.Fatalf("Get should return nil for expired entry, got: %v", v)
	}
	if l.Len() != 0 {
		t.Fatalf("Len should be 0 after Get removes expired entry, got: %v", l.Len())
	}
}

// Test that Contains removes expired entries from the cache (Len decreases)
func TestLRU_Contains_RemovesExpired(t *testing.T) {
	l, err := NewLRUWithExpire(10, 10*time.Millisecond, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	l.Add(1, "value")
	if l.Len() != 1 {
		t.Fatalf("bad len after add: %v", l.Len())
	}

	time.Sleep(20 * time.Millisecond) // let it expire

	if l.Contains(1) {
		t.Fatalf("Contains should return false for expired entry")
	}
	if l.Len() != 0 {
		t.Fatalf("Len should be 0 after Contains removes expired entry, got: %v", l.Len())
	}
}

// Test that Peek removes expired entries from the cache (Len decreases)
func TestLRU_Peek_RemovesExpired(t *testing.T) {
	l, err := NewLRUWithExpire(10, 10*time.Millisecond, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	l.Add(1, "value")
	if l.Len() != 1 {
		t.Fatalf("bad len after add: %v", l.Len())
	}

	time.Sleep(20 * time.Millisecond) // let it expire

	v, ok := l.Peek(1)
	if ok {
		t.Fatalf("Peek should return false for expired entry, got: %v", v)
	}
	if l.Len() != 0 {
		t.Fatalf("Len should be 0 after Peek removes expired entry, got: %v", l.Len())
	}
}

// Test that Keys() handles mixed expired/non-expired entries and returns correct count
func TestLRU_Keys_MixedExpired(t *testing.T) {
	l, err := NewLRUWithExpire(10, 100*time.Millisecond, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Add entries with short expiration
	l.AddEx(1, "a", 10*time.Millisecond)
	l.AddEx(2, "b", 500*time.Millisecond)
	l.AddEx(3, "c", 10*time.Millisecond)
	l.AddEx(4, "d", 500*time.Millisecond)

	time.Sleep(50 * time.Millisecond) // entries 1 and 3 expire

	// Keys should return expired entries too (they're still in data structures)
	keys := l.Keys()
	if len(keys) != 4 {
		t.Fatalf("Keys() count before Get() cleanup, expected 4, got: %v", len(keys))
	}
	if l.Len() != 4 {
		t.Fatalf("Len before cleanup, expected 4, got: %v", l.Len())
	}

	// Access expired entries via Get — they should be removed
	l.Get(1)
	l.Get(3)

	if l.Len() != 2 {
		t.Fatalf("Len after Get() cleanup, expected 2, got: %v", l.Len())
	}

	// Keys() should now return only non-expired entries
	keys = l.Keys()
	if len(keys) != 2 {
		t.Fatalf("Keys() after cleanup, expected 2, got: %v (keys: %v)", len(keys), keys)
	}
}

// Test that Keys() doesn't panic after extensive mixed operations
func TestLRU_Keys_Stress(t *testing.T) {
	l, err := NewLRU(100, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Fill the cache
	for i := 0; i < 200; i++ {
		l.Add(i, i)
	}

	// Mix of operations: Remove some, Get some (reorder), Add more
	for i := 0; i < 100; i++ {
		l.Remove(i * 2) // remove evens from 0-198
	}
	for i := 100; i < 150; i++ {
		l.Get(i) // touch some, reorder
	}
	for i := 200; i < 250; i++ {
		l.Add(i, i) // add new
	}

	// Keys() should never panic regardless of internal state
	keys := l.Keys()
	if len(keys) != l.Len() {
		t.Fatalf("Keys() count (%d) != Len() (%d)", len(keys), l.Len())
	}

	// Verify keys contain no duplicates
	seen := make(map[interface{}]bool)
	for _, k := range keys {
		if seen[k] {
			t.Fatalf("duplicate key in Keys(): %v", k)
		}
		seen[k] = true
	}
}

// Test that Keys() returns empty slice, not nil, for empty cache
func TestLRU_Keys_Empty(t *testing.T) {
	l, err := NewLRU(10, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	keys := l.Keys()
	if keys == nil {
		t.Fatal("Keys() should return non-nil for empty cache")
	}
	if len(keys) != 0 {
		t.Fatalf("Keys() should return 0 for empty cache, got: %v", len(keys))
	}
}
