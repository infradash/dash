package dash

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrUnknownState    error = errors.New("unknown-state")
	ErrStateNotAllowed error = errors.New("state-not-allowed")
	ErrInvalidState    error = errors.New("invalid-state")
)

type State interface {
	String() string
	Equals(State) bool
}

type StateTransitions interface {
	Next(State) ([]State, bool)
}

type FsmState struct {
	Id      int       `json:"id"`
	State   State     `json:"state"`
	Started time.Time `json:"started"`
	Expiry  time.Time `json:"expiry"`
	Message string    `json:"message"`
	Error   error     `json:"error"`

	fsm   *Fsm // parent
	timer *time.Timer
}

func (this *FsmState) String() string {
	e := "no-error"
	if this.Error != nil {
		e = this.Error.Error()
	}
	return fmt.Sprintf("%03d,%s,%s,%s,%s,%s",
		this.Id,
		this.State.String(),
		this.Started,
		this.Expiry,
		this.Message,
		e)
}

type Fsm struct {
	Definition StateTransitions

	CustomData interface{} `json:"custom_data,omitempty"`
	History    []*FsmState `json:"history"`

	Expiration <-chan *FsmState `json:"-"`
	Complete   <-chan *FsmState `json:"-"`

	pending    *time.Timer
	expiration chan<- *FsmState
	complete   chan<- *FsmState

	sync sync.Mutex
}

func NewFsm(def StateTransitions, initial State) *Fsm {
	s := new(FsmState)
	s.Id = 0
	s.State = initial
	s.Started = time.Now()
	exp := make(chan *FsmState, 0)
	complete := make(chan *FsmState, 0)
	fsm := &Fsm{
		Definition: def,
		History:    []*FsmState{s},
		Expiration: exp,
		expiration: exp,
		Complete:   complete,
		complete:   complete,
	}

	s.fsm = fsm
	return fsm
}

func (this *Fsm) Current() *FsmState {
	if len(this.History) > 0 {
		return this.History[len(this.History)-1]
	}
	return nil
}

func (this *Fsm) check_transition(current, next State) error {
	if allowed, has := this.Definition.Next(current); has {
		for _, s := range allowed {
			if s.Equals(next) {
				return nil
			}
		}
	}
	return ErrStateNotAllowed
}

// Transition to the next state specified.  Once the state is obtained, the deadline can be set.
func (this *Fsm) Next(next State, message string, observed error) (*FsmState, error) {
	this.sync.Lock()
	defer this.sync.Unlock()

	current := this.History[len(this.History)-1]
	if err := this.check_transition(current.State, next); err != nil {
		return nil, err
	}

	s := &FsmState{
		Id:      len(this.History),
		State:   next,
		Message: message,
		Error:   observed,
		Started: time.Now(),
		fsm:     this,
	}

	// cancel pending timer because we are transitioning so we don't get event on Expiration
	if this.pending != nil {
		this.pending.Stop()
	}

	// Non blocking send to Complete
	select {
	case this.complete <- current:
	default:
	}

	// Append only for state changes
	this.History = append(this.History, s)

	return s, nil
}

// Sets the start time and the expiry.  This also starts a timer
func (this *FsmState) SetDeadline(duration time.Duration) error {
	this.fsm.sync.Lock()
	defer this.fsm.sync.Unlock()

	// Check to see if it's current state
	if this.fsm.Current() != this {
		return ErrInvalidState
	}

	this.Started = time.Now()
	this.Expiry = this.Started.Add(duration)
	this.timer = time.AfterFunc(duration, func() {
		// Timeup: non blocking send
		select {
		case this.fsm.expiration <- this:
		default:
		}
		this.fsm.sync.Lock()
		this.fsm.pending = nil
		this.fsm.sync.Unlock()
	})
	this.fsm.pending = this.timer

	return nil
}
