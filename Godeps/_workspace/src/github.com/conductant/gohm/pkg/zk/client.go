package zk

import (
	"github.com/conductant/gohm/pkg/store"
	"github.com/golang/glog"
	"github.com/samuel/go-zookeeper/zk"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

type client struct {
	url     url.URL
	conn    *zk.Conn
	servers []string
	timeout time.Duration
	events  chan Event

	ephemeral        map[string]*Node
	ephemeral_add    chan *Node
	ephemeral_remove chan string

	retry      chan *Node
	retry_stop chan int
	stop       chan int

	running bool

	watch_stops_chan chan chan int
	watch_stops      map[chan int]bool

	shutdown chan int
	close    store.Dispose
}

func (this *client) on_connect() {
	for _, n := range this.ephemeral {
		this.retry <- n
	}
}

// ephemeral flag here is user requested.
func (this *client) trackEphemeral(zn *Node, ephemeral bool) {
	if ephemeral || (zn.Stats != nil && zn.Stats.EphemeralOwner > 0) {
		glog.Infoln("ephemeral-add:", "path=", zn.Path)
		this.ephemeral_add <- zn
	}
}

func (this *client) untrackEphemeral(path string) {
	this.ephemeral_remove <- path
}

func Connect(servers []string, timeout time.Duration) (*client, error) {
	conn, events, err := zk.Connect(servers, timeout)
	glog.Infoln("Connect to zk:", "conn=", conn, "events=", events, "err=", err)
	if err != nil {
		return nil, err
	}
	u, _ := url.Parse("zk://" + strings.Join(servers, ","))
	zz := &client{
		url:              *u,
		conn:             conn,
		servers:          servers,
		timeout:          timeout,
		events:           make(chan Event, 4096),
		stop:             make(chan int),
		ephemeral:        map[string]*Node{},
		ephemeral_add:    make(chan *Node),
		ephemeral_remove: make(chan string),
		retry:            make(chan *Node, 1024),
		retry_stop:       make(chan int),
		watch_stops:      make(map[chan int]bool),
		watch_stops_chan: make(chan chan int),
		shutdown:         make(chan int),
	}

	go func() {
		<-zz.shutdown
		zz.doShutdown()
		glog.Infoln("Shutdown complete.")
	}()

	go func() {
		defer glog.Infoln("ZK watcher cache stopped.")
		for {
			watch_stop, open := <-zz.watch_stops_chan
			if !open {
				return
			}
			zz.watch_stops[watch_stop] = true
		}
	}()
	go func() {
		defer glog.Infoln("ZK ephemeral cache stopped.")
		for {
			select {
			case add, open := <-zz.ephemeral_add:
				if !open {
					return
				}
				zz.ephemeral[add.Path] = add
				glog.Infoln("ephemeral-add: Path=", add.Path, "Value=", string(add.Value))

			case remove, open := <-zz.ephemeral_remove:
				if !open {
					return
				}
				if _, has := zz.ephemeral[remove]; has {
					delete(zz.ephemeral, remove)
					glog.Infoln("ephemeral-remove: Path=", remove)
				}
			}
		}
	}()
	go func() {
		defer glog.Infoln("ZK event loop stopped")
		for {
			select {
			case evt := <-events:
				glog.Infoln("zk-event-chan:", evt)
				switch evt.State {
				case StateExpired:
					glog.Warningln("ZK state expired --> sent by server on reconnection.")
					// This is actually connected, despite the state name, because the server
					// sends this event on reconnection.
					zz.on_connect()
				case StateHasSession:
					zz.on_connect()
				case StateDisconnected:
					glog.Warningln("ZK state disconnected")
				}
				zz.events <- Event{Event: evt}
			case <-zz.stop:
				return
			}
		}
	}()
	go func() {
		defer glog.Infoln("ZK ephemeral resync loop stopped")
		for {
			select {
			case r := <-zz.retry:
				if r != nil {
					_, err := zz.CreateNode(r.Path, r.Value, true)
					switch err {
					case nil, ErrNodeExists:
						glog.Infoln("emphemeral-resync: Key=", r.Path, "retry ok.")
						zz.events <- Event{Event: zk.Event{Path: r.Path}, Action: "Ephemeral-Retry", Note: "retry ok"}
					default:
						glog.Infoln("emphemeral-resync: Key=", r.Path, "Err=", err, "retrying.")
						select {
						case zz.retry <- r:
							glog.Infoln("emphemeral-resync:", r.Path, "submitted")
							select {
							case zz.events <- Event{
								Event:  zk.Event{Path: r.Path},
								Action: "ephemeral-resync",
								Note:   "retrying"}:
							}
						default:
							glog.Warningln("ephemeral-resync: dropped object", r.Path)
						}
					}
				}
			case <-zz.retry_stop:
				return
			}
		}
	}()

	glog.Infoln("Connected to zk:", servers)
	return zz, nil
}

func (this *client) check() error {
	if this.conn == nil {
		return ErrNotConnected
	}
	return nil
}

func (this *client) Events() <-chan Event {
	return this.events
}

func (this *client) Close() error {
	ok := true
	if this.close != nil {
		// If this is used in connection with a registry cache. notify it we are done.
		// The protocol here is to first propose and wait for ok to shutdown
		glog.Infoln("Propose to close", this.close.Propose())
		this.close.Propose() <- this
		glog.Infoln("Waiting for accept")
		ok = <-this.close.Accept()
		glog.Infoln("Got accept to close=", ok)
	}
	if ok {
		this.shutdown <- 1
		// wait for a close
		<-this.shutdown
	}
	return nil
}

func (this *client) doShutdown() {
	glog.Infoln("Shutting down...")

	close(this.ephemeral_add)
	close(this.ephemeral_remove)

	close(this.stop)
	close(this.retry_stop)

	for w, _ := range this.watch_stops {
		select {
		case w <- 0:
		default:
		}
		//close(w)  TODO - FIX THIS   http://blog.golang.org/pipelines
	}
	close(this.watch_stops_chan)

	this.conn.Close()
	this.conn = nil

	close(this.shutdown)
}

func (this *client) Reconnect() error {
	p, err := Connect(this.servers, this.timeout)
	if err != nil {
		return err
	} else {
		this = p
		return nil
	}
}

func (this *client) GetNode(path string) (*Node, error) {
	if err := this.check(); err != nil {
		return nil, err
	}

	exists, _, err := this.conn.Exists(path)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotExist
	}
	value, stats, err := this.conn.Get(path)
	if err != nil {
		return nil, err
	}
	return &Node{Path: path, Value: value, Stats: stats, client: this}, nil
}

func (this *client) WatchOnce(path string, f func(Event)) (chan<- bool, error) {
	if err := this.check(); err != nil {
		return nil, err
	}
	_, _, event_chan, err := this.conn.ExistsW(path)
	if err != nil {
		return nil, err
	}
	return runWatch(path, f, event_chan)
}

func (this *client) WatchOnceChildren(path string, f func(Event)) (chan<- bool, error) {
	if err := this.check(); err != nil {
		return nil, err
	}

	_, _, event_chan, err := this.conn.ChildrenW(path)
	switch {

	case err == ErrNotExist:
		_, _, event_chan0, err0 := this.conn.ExistsW(path)
		if err0 != nil {
			return nil, err0
		}
		// First watch for creation
		// Use a common stop
		stop1 := make(chan bool)
		_, err1 := runWatch(path, func(e Event) {
			if e.Type == zk.EventNodeCreated {
				if _, _, event_chan2, err2 := this.conn.ChildrenW(path); err2 == nil {
					// then watch for children
					runWatch(path, f, event_chan2, stop1)
				}
			}
		}, event_chan0, stop1)
		return stop1, err1

	case err == nil:
		return runWatch(path, f, event_chan)

	default:
		return nil, err
	}
}

// Continuously watch a path, optional callbacks for errors.
func (this *client) Watch(path string, f func(Event), errs ...chan<- error) (chan<- int, <-chan error, error) {
	return this.watch(func() (<-chan zk.Event, error) {
		_, _, ch, err := this.conn.ExistsW(path)
		return ch, err
	}, path, f, errs...)
}

// Watch the children under a path. Only a change in the count of children will result in an event.
// A mutation on the value of a child will not trigger an event.  A creation or deletion of a child node will.
func (this *client) WatchChildren(path string, f func(Event), errs ...chan<- error) (chan<- int, <-chan error, error) {
	return this.watch(func() (<-chan zk.Event, error) {
		if _, _, ch, err := this.conn.ChildrenW(path); err == ErrNotExist {
			_, _, ch, err := this.conn.ExistsW(path)
			return ch, err
		} else {
			return ch, err
		}
	}, path, f, errs...)
}

type getChan func() (<-chan zk.Event, error)
type callBack func(Event)

func (this *client) watch(cf getChan, p string, f callBack, errs ...chan<- error) (chan<- int, <-chan error, error) {
	if err := this.check(); err != nil {
		return nil, nil, err
	}

	event_chan, err := cf()
	if err != nil {
		for _, a := range errs {
			a <- err
		}
		return nil, nil, err
	}

	stop := make(chan int)
	stopped := make(chan error)

	this.watch_stops_chan <- stop

	// Use a buffered channel and a separate goroutine to dispatch the event to the user provided callback.
	// The aim here is to reduce the lag between re-subscription of the watch.  This however, can make
	// timing unpredicatble since the user's callback is executed in a separate thread.
	bufferedChan := make(chan *zk.Event, 64)
	go func() {

		defer func() {
			close(bufferedChan)
			stopped <- nil
		}()

		for {
			event := <-bufferedChan
			if event == nil {
				glog.Infoln("watch-callback-stop")
				return
			}
			glog.Infoln("watch-event-process", "path=", p, "type=", event.Type, "state=", event.State)
			switch event.State {
			case zk.StateExpired:
				for _, a := range errs {
					a <- ErrSessionExpired
				}
			case zk.StateDisconnected:
				for _, a := range errs {
					a <- ErrConnectionClosed
				}
			default:
				f(Event{Event: *event})
			}
		}
	}()

	go func() {
		glog.Infoln("watch: Started watch on", "path=", p)
		for {
			select {
			case event := <-event_chan:

				// Buffered channel should not block unless it's full.
				bufferedChan <- &event
				glog.Infoln("watch-event-dispatched", "path=", p, "type=", event.Type, "state=", event.State)

				for { // Retry / resubscribe loop
					success := false
					event_chan, err = cf()
					if err == nil {
						success = true
						this.events <- Event{Event: zk.Event{Path: p}, Action: "Watch-Retry", Note: "retry ok"}
						glog.Infoln("watch-retry: Continue watching", p)
					}
					if success {
						break
					} else {
						for _, a := range errs {
							select { // non blocking send
							case a <- err:
							default:
							}
						}
						this.events <- Event{Event: zk.Event{Path: p}, Action: "watch-retry", Note: "retrying"}
					}
				}

			case <-stop:
				glog.Infoln("watch: Watch terminated:", "path=", p)
				// Send a nil to the processing goroutine to stop it.
				bufferedChan <- nil
				return
			}
		}
	}()
	return stop, stopped, nil
}

// Creates a new node.  If node already exists, error will be returned.
func (this *client) CreateNode(path string, value []byte, ephemeral bool) (*Node, error) {
	if err := this.check(); err != nil {
		return nil, err
	}
	// Make sure all parents exist - they can't be ephemeral
	err := this.createParents(path)
	if err != nil {
		return nil, err
	}
	return this.createNode(path, value, ephemeral)
}

func listParents(path string) []string {
	p := path
	if p[0:1] != "/" {
		p = "/" + path // Must begin with /
	}
	pp := strings.Split(p, "/")
	t := []string{}
	root := ""
	for _, x := range pp[1:] {
		z := root + "/" + x
		root = z
		t = append(t, z)
	}
	return t
}

func (this *client) createParents(path string) error {
	dir := filepath.Dir(path)
	if dir == "." {
		return nil
	}
	for _, p := range listParents(dir) {
		exists, _, err := this.conn.Exists(p)
		if err != nil {
			return err
		}
		if !exists {
			_, err := this.createNode(p, []byte{}, false)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Assumes all parent nodes have been created.
func (this *client) createNode(path string, value []byte, ephemeral bool) (*Node, error) {
	key := path
	flags := int32(0)
	if ephemeral {
		flags = int32(zk.FlagEphemeral)
	}
	acl := zk.WorldACL(zk.PermAll) // TODO - PermAll permission
	p, err := this.conn.Create(key, value, flags, acl)
	if err != nil {
		return nil, err
	}
	zn := &Node{Path: p, Value: value, client: this}
	this.trackEphemeral(zn, ephemeral)
	return this.GetNode(p)
}

// Sets the node value, creates if not exists.
func (this *client) PutNode(key string, value []byte, ephemeral bool) (*Node, error) {
	if ephemeral {
		n, err := this.CreateNode(key, value, true)
		return n, err
	}
	n, err := this.GetNode(key)
	switch err {
	case nil:
		return n, n.Set(value)
	case ErrNotExist:
		n, err = this.CreateNode(key, value, false)
		if err != nil {
			return nil, err
		} else {
			return n, nil
		}
	default:
		return nil, err
	}
}

func (this *client) DeleteNode(path string) error {
	if err := this.check(); err != nil {
		return err
	}
	this.untrackEphemeral(path)
	return this.conn.Delete(path, -1)
}

func runWatch(path string, f func(Event), event_chan <-chan zk.Event, optionalStop ...chan bool) (chan bool, error) {
	if f == nil {
		return nil, nil
	}

	stop := make(chan bool, 1)
	if len(optionalStop) > 0 {
		stop = optionalStop[0]
	}

	go func() {
		// Note ZK only fires once and after that we need to reschedule.
		// With this api this may mean we get a new event channel.
		// Therefore, there's no point looping in here for more than 1 event.
		select {
		case event := <-event_chan:
			f(Event{Event: event})
		case b := <-stop:
			if b {
				glog.Infoln("watch-terminated:", "path=", path)
				return
			}
		}
	}()
	return stop, nil
}
