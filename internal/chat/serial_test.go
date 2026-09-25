package chat

import (
	"sync"
	"testing"
	"time"
)

// counter records how many sections are live at once, so a chain that fails to
// serialize reads as a peak above one instead of a silent race.
type counter struct {
	mu     sync.Mutex
	active int
	high   int
}

func (c *counter) enter() {
	c.mu.Lock()
	c.active++
	if c.active > c.high {
		c.high = c.active
	}
	c.mu.Unlock()
}

func (c *counter) leave() {
	c.mu.Lock()
	c.active--
	c.mu.Unlock()
}

func (c *counter) max() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.high
}

// gated waits for one piece of work's turn, runs it, and releases the next.
// Every test below goes through it so that what is under test is the chain
// from wait and not a per-test reimplementation of it.
func gated(gate <-chan struct{}, release func(), fn func()) {
	if gate != nil {
		<-gate
	}
	defer release()
	fn()
}

// start queues one piece of work behind its key's chain, exactly as Run does
// for one accepted message.
func start(s *serial, wg *sync.WaitGroup, key string, fn func()) {
	gate, release := s.wait(key)
	wg.Go(func() { gated(gate, release, fn) })
}

// TestSerialKeepsOneConversationInOrder proves the property a plain lock does
// not give: two pieces of work chained on one key run one after the other and
// in the order wait was called, which is arrival order. The order is decided
// by wait, so the test calls it synchronously in sequence exactly as the
// receive callback does, and only then runs the work concurrently. Reversing
// the chain would let the second question be answered first, which is the
// defect this type exists to prevent.
func TestSerialKeepsOneConversationInOrder(t *testing.T) {
	s := newSerial()
	const key = "matrix:!room:example.org"
	var mu sync.Mutex
	var order []string

	record := func(name string) func() {
		return func() {
			mu.Lock()
			order = append(order, name)
			mu.Unlock()
		}
	}
	gateA, releaseA := s.wait(key)
	gateB, releaseB := s.wait(key)

	var wg sync.WaitGroup
	wg.Go(func() { gated(gateB, releaseB, record("second")) })
	wg.Go(func() { gated(gateA, releaseA, record("first")) })
	wg.Wait()

	if len(order) != 2 || order[0] != "first" || order[1] != "second" {
		t.Fatalf("ran in order %v, want [first second]", order)
	}
}

// TestSerialSerializeSameKey proves two pieces of work sharing one key never
// overlap. Mutual exclusion is a negative property, so it is observed rather
// than signalled: the first holds the section, the second is given a grace
// window to enter, and entering inside that window fails the test. The counter
// records the peak, so a broken chain reports concurrency rather than only a
// timeout.
func TestSerialSerializeSameKey(t *testing.T) {
	s := newSerial()
	const key = "matrix:!room:example.org"
	c := &counter{}
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	var wg sync.WaitGroup

	work := func() {
		c.enter()
		entered <- struct{}{}
		<-release
		c.leave()
	}

	start(s, &wg, key, work)
	<-entered

	start(s, &wg, key, work)

	select {
	case <-entered:
		t.Fatal("a second message entered while the first still held the key")
	case <-time.After(100 * time.Millisecond):
	}

	close(release)
	wg.Wait()
	if c.max() != 1 {
		t.Fatalf("peak concurrent sections = %d, want 1", c.max())
	}
}

// TestSerialAllowsDifferentKeys proves two conversations do not wait on each
// other: both enter before either is released, which one shared chain could
// not allow.
func TestSerialAllowsDifferentKeys(t *testing.T) {
	s := newSerial()
	entered := make(chan string, 2)
	release := make(chan struct{})
	var wg sync.WaitGroup

	work := func(key string) func() {
		return func() {
			entered <- key
			<-release
		}
	}

	gateA, releaseA := s.wait("matrix:!room-a:example.org")
	gateB, releaseB := s.wait("matrix:!room-b:example.org")
	wg.Go(func() { gated(gateA, releaseA, work("a")) })
	wg.Go(func() { gated(gateB, releaseB, work("b")) })

	for range 2 {
		select {
		case <-entered:
		case <-time.After(time.Second):
			t.Fatal("two different keys could not run at the same time")
		}
	}
	close(release)
	wg.Wait()
}
