package dash

import (
	"errors"
	. "gopkg.in/check.v1"
	"testing"
	"time"
)

const (
	Starting containerState = iota
	Running                 // ready running
	Stopping                // initiated stop
	Stopped                 // stopped
	Failed                  // uninitiated / unexpected stop
	Removed                 // removed
)

var (
	container_state_labels = map[containerState]string{
		Starting: "fsm:starting",
		Running:  "fsm:running",
		Stopping: "fsm:stopping",
		Stopped:  "fsm:stopped",
		Failed:   "fsm:failed",
		Removed:  "fsm:removed",
	}

	container = containerFsm{
		Starting: []State{Running, Failed, Stopping},
		Running:  []State{Running, Failed, Stopping, Stopped},
		Stopping: []State{Failed, Stopped},
		Stopped:  []State{Removed},
		Failed:   []State{Removed},
	}
)

type containerState int
type containerFsm map[containerState][]State

func (this containerState) String() string {
	return container_state_labels[this]
}

func (this containerState) Equals(that State) bool {
	if typed, ok := that.(containerState); ok {
		return typed == this
	}
	return false
}

func (this containerFsm) Instance(initial containerState) *Fsm {
	return NewFsm(this, initial)
}

func (this containerFsm) Next(s State) (v []State, c bool) {
	if typed, ok := s.(containerState); ok {
		v, c = this[typed]
		return
	}
	return nil, false
}

func TestFsm(t *testing.T) { TestingT(t) }

type TestSuiteFsm struct {
}

var _ = Suite(&TestSuiteFsm{})

func (suite *TestSuiteFsm) SetUpSuite(c *C) {
}

func (suite *TestSuiteFsm) TearDownSuite(c *C) {
}

func (suite *TestSuiteFsm) TestFsm(c *C) {

	// This is the case when we are triggering a container run
	fsm1 := container.Instance(Starting)
	c.Assert(fsm1.Current().State, Equals, Starting)

	// Now we get an event from docker watch...
	s1, err := fsm1.Next(Running, "is getting ready...", nil)
	c.Assert(err, Equals, nil)

	c1 := fsm1.Current()
	c.Assert(s1, Equals, c1)

	// We noticed a stop
	s1, err = fsm1.Next(Stopped, "Container stopped", nil)
	c.Assert(err, Equals, nil)
	c.Assert(fsm1.Current().State, Equals, Stopped)

	// Then it's removed
	s1, err = fsm1.Next(Removed, "Removed", nil)
	c.Assert(err, Equals, nil)
	c.Assert(s1.State, Equals, Removed)

	// This is the case when we discover something that's already running
	fsm2 := container.Instance(Running)
	_, err = fsm2.Next(Removed, "bad transition", nil)
	c.Assert(err, Not(Equals), nil)
	c.Assert(fsm2.Current().State, Equals, Running)

	// we stop it
	_, err = fsm2.Next(Stopping, "Stopping the container", nil)
	c.Assert(err, Equals, nil)

	_, err = fsm2.Next(Failed, "Got error", errors.New("bad-thing"))
	c.Assert(err, Equals, nil)
	c.Assert(fsm2.Current().State, Equals, Failed)
	c.Log("final state=", fsm2.Current().State)
}

func (suite *TestSuiteFsm) TestFsmWithDeadline(c *C) {

	// This is the case when we are triggering a container run
	// Id = 0
	fsm1 := container.Instance(Starting)
	c.Assert(fsm1.Current().State, Equals, Starting)

	state := fsm1.Current()
	c.Assert(state, Not(Equals), nil)

	err := state.SetDeadline(1 * time.Second)
	c.Assert(err, Equals, nil)

	done := make(chan int)
	go func() {
		select {
		// Wait for deadline
		case s := <-fsm1.Expiration:
			c.Assert(s.Id, Equals, 0)
			c.Log("Expired: ", s)
			done <- s.Id
		}
	}()

	// do something else here.

	// now wait
	v := <-done
	c.Assert(v, Equals, 0)

	// Now we advance
	c.Log("Running container... with deadline")

	// Id = 1
	fsm1.Next(Running, "Running container now... but with deadline", nil)
	fsm1.Current().SetDeadline(10 * time.Second)

	complete := make(chan int)
	go func() {
		c.Log("Starting to wait for running to finish")

		select {
		// Wait for cancel
		case s := <-fsm1.Complete:
			c.Assert(s.Id, Equals, 1) // 1 finished
			c.Log("Completed: ", s)
			complete <- s.Id
		}
	}()

	time.Sleep(2)
	// But we receive a crash signal here
	c.Log("Crash observed")

	// Id = 2
	fsm1.Next(Failed, "Ooops, crashed", errors.New("some-error"))
	v = <-complete
	c.Assert(v, Equals, 1)
	c.Assert(fsm1.Current().State, Equals, Failed)
}
