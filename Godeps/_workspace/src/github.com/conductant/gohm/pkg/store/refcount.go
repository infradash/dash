package store

import (
	"fmt"
	"github.com/golang/glog"
	"sync"
)

type Key interface{}
type Object interface{}
type AllocatorFunc func(Dispose) (Key, Object, error)
type KeyFunc func(Object) Key

// Protocol between implementation and the store.  The implementation is expected to
// send to Propose and then listen for a True on Accept.  If true then the implementation
// can actually close and dispose the resources.  This allows the central registry to
// implement shared connections and things like reference counting.
type Dispose interface {
	Propose() chan<- Object
	Accept() <-chan bool
}

type dispose struct {
	propose chan Object
	accept  chan bool
}

func (this *dispose) Propose() chan<- Object {
	return this.propose
}
func (this *dispose) Accept() <-chan bool {
	return this.accept
}

// Reference counter for the given regstry
type reference struct {
	key     Key
	object  Object
	count   int
	dispose *dispose
}

type cache map[Key]*reference

// The implementation of the referencing counting store.
type RefCountStore struct {
	lock             sync.Mutex
	cache            cache
	proposeToDispose chan Object
	objectReferenced chan Key
	objectCreated    chan *reference
	done             chan int
	stopped          chan error
	keyFunc          KeyFunc
}

// The referencing counting store allows a client to delegate the referencing counting and
// tracking to this store via the `Dispose` interface and Track function.  This allows
// the client to be arbitrarily complex in terms of object creation and deallocation.
func NewRefCountStore(kf KeyFunc) *RefCountStore {
	return &RefCountStore{
		cache:            cache{},
		objectCreated:    make(chan *reference),
		objectReferenced: make(chan Key),
		proposeToDispose: make(chan Object),
		done:             make(chan int),
		stopped:          make(chan error),
		keyFunc:          kf,
	}
}

func (this cache) add(key Key, ref *reference) {
	if r, has := this[key]; !has {
		this[key] = ref
	} else {
		// Some how we allocated another resource with the same key.
		panic(fmt.Errorf("Key collision: existing=%v, new=%v", r, ref))
	}
}

func (this cache) get(key Key) *reference {
	// Without locking but check to make sure we don't return something that will be garbage collected.
	if ref, has := this[key]; has && ref.count > 0 {
		return ref
	}
	return nil
}

func (this cache) remove(key Key) {
	delete(this, key)
}

// Track an object references. The alloc function takes a dispose
// that it can later on use to notify when the object is about to be disposed and
// get approval for actual disposal.  The alloc function should return the object to be
// reference counted.
func (this *RefCountStore) allocate(alloc AllocatorFunc) (Object, error) {
	d := &dispose{
		propose: this.proposeToDispose,
		accept:  make(chan bool, 1),
	}
	key, obj, err := alloc(d)
	if err != nil {
		return nil, err
	} else {
		ref := &reference{
			dispose: d,
			count:   1,
			key:     key,
			object:  obj,
		}
		this.objectCreated <- ref
		return obj, nil
	}
}

func (this *RefCountStore) Get(key Key, alloc AllocatorFunc) (Object, error) {
	ref := this.cache.get(key)
	if ref != nil {
		this.objectReferenced <- ref.key
		return ref.object, nil
	} else {
		// Allocate
		return this.allocate(alloc)
	}
}

func (this *RefCountStore) Stop(code int) <-chan error {
	this.done <- code
	return this.stopped
}

func (this *RefCountStore) Start() *RefCountStore {
	// Go routine here to keep track of registries by protocol and host string.  For a given
	// scheme/host combination, we are using the same object, unless it's been closed, which
	// will then cause the object to report its closing and be removed from cache.
	go func() {
		for {
			select {
			case obj := <-this.proposeToDispose:
				key := this.keyFunc(obj)
				ref := this.cache.get(key)
				if ref != nil {
					ref.count--
					glog.V(500).Infoln("Propose to dispose:", "key=", key, "ref=", ref.count)
					if ref.count == 0 {
						this.cache.remove(key)
						ref.dispose.accept <- true
						glog.V(500).Infoln("Removed object key=", key)
					} else {
						ref.dispose.accept <- false // do not dispose resources. others are using.
					}
				} else {
					panic("shouldn't be here!")
				}
			case ref := <-this.objectCreated:
				this.cache.add(ref.key, ref)
			case key := <-this.objectReferenced:
				ref := this.cache.get(key)
				if ref != nil {
					ref.count++
					glog.V(500).Infoln("Referencing object:", "key=", key, "ref=", ref.count)
				} else {
					panic("shouldn't be here")
				}
			case <-this.done:
				break
			}
		}
		this.stopped <- nil
	}()
	return this
}
