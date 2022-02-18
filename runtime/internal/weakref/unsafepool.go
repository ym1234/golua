package weakref

import (
	"log"
	"runtime"
	"sort"
	"sync"
	"unsafe"
)

//
// Unsafe Pool implementation
//

// UnsafePool is an implementation of Pool that makes every effort to let
// values be GCed when they are only reachable via WeakRefs.  It relies on
// casting interface{} to unsafe pointers and back again, which would break if
// Go were to have a moving GC.
type UnsafePool struct {
	mx            sync.Mutex           // Used to synchronize access to weakrefs, pendingVals, pendingOrders.
	weakrefs      map[uintptr]*weakRef //
	pending       sortabelVals         // Values pending Lua finalization
	lastMarkOrder int                  // this is to sort values by reverse order of mark for finalize
}

var _ Pool = &UnsafePool{}

// NewUnsafePool returns a new *UnsafeWeakRefPool ready to be used.
func NewUnsafePool() *UnsafePool {
	return &UnsafePool{weakrefs: make(map[uintptr]*weakRef)}
}

// Get returns a *WeakRef for v if possible.
func (p *UnsafePool) Get(iface interface{}) WeakRef {
	p.mx.Lock()
	defer p.mx.Unlock()
	return p.get(iface)
}

// Returns a *WeakRef for iface, not thread safe, only call when you have the
// pool lock.
func (p *UnsafePool) get(iface interface{}) *weakRef {
	w := getwiface(iface)
	id := w.id()
	r := p.weakrefs[id]
	if r == nil {
		runtime.SetFinalizer(iface, p.addPendingGC)
		r = &weakRef{
			w:    getwiface(iface),
			pool: p,
		}
		p.weakrefs[id] = r
	}
	return r
}

// Mark marks v for finalizing, i.e. when v is garbage collected, its finalizer
// should be run.  It only takes effect if v can have a weak ref.
func (p *UnsafePool) Mark(iface interface{}) {
	p.mx.Lock()
	defer p.mx.Unlock()
	p.lastMarkOrder++
	p.get(iface).markOrder = p.lastMarkOrder
}

// ExtractDeadMarked returns the set of values which are being garbage collected
// and need their finalizer running, in the order that they should be run.  The
// caller of this function has the responsibility to run all the finalizers. The
// values returned are removed from the pool and their weak refs are
// invalidated.
func (p *UnsafePool) ExtractDeadMarked() []interface{} {
	p.mx.Lock()
	pending := p.pending
	if pending == nil {
		// This is the common case, so it's worth exiting early
		p.mx.Unlock()
		return nil
	}
	p.pending = nil
	p.mx.Unlock()
	// Lua wants to run finalizers in reverse order
	sort.Sort(pending)
	log.Printf("Extract Dead %d\n", len(pending))
	return runPrefinalizers(pending.vals())
}

// ExtractAllMarked returns all the values that have been marked for finalizing,
// whether they are dead or not.  This is useful e.g. when closing a runtime, to
// run all pending finalizers.
func (p *UnsafePool) ExtractAllMarked() []interface{} {
	p.mx.Lock()
	marked := p.pending
	for _, r := range p.weakrefs {
		if r.markOrder > 0 {
			iface := r.w.iface()
			marked = append(marked, orderedVal{
				val:   iface,
				order: r.markOrder,
			})

			r.markOrder = 0
			// We don't want the finalizer to be triggered anymore, but more
			// important the finalizer is holding a reference to the pool
			// (although that may not affect its reachability?)
			runtime.SetFinalizer(iface, nil)
		}
	}
	p.pending = nil
	p.mx.Unlock()
	// Sort in reverse order
	sort.Sort(marked)
	return runPrefinalizers(marked.vals())
}

// This is the finalizer that Go runs on values added to the pool when they
// become unreachable.
func (p *UnsafePool) addPendingGC(iface interface{}) {
	p.mx.Lock()
	defer p.mx.Unlock()
	id := getwiface(iface).id()
	r := p.weakrefs[id]
	if r == nil {
		return
	}
	if r.status == wrResurrected {
		r.status = wrAlive
		runtime.SetFinalizer(iface, p.addPendingGC)
		return
	}
	r.status = wrDead
	if r.markOrder > 0 {
		p.pending = append(p.pending, orderedVal{
			val:   iface,
			order: r.markOrder,
		})
	}
	delete(p.weakrefs, id)
}

//
// WeakRef implementation for UnsafePool
//

type weakRef struct {
	w         wiface // encodes the value the weak ref refers to
	markOrder int    // positive if the value was marked with WeakRefPool.Mark()
	status    wrStatus

	// Needed to sync with the Go finalizers which run in their own goroutine.
	pool *UnsafePool
}

var _ WeakRef = &weakRef{}

// Value returns the value this weak ref refers to if it is still alive, else
// returns NilValue.
func (r *weakRef) Value() interface{} {
	r.pool.mx.Lock()
	defer r.pool.mx.Unlock()
	if r.status == wrDead {
		return nil
	}
	r.status = wrResurrected
	return r.w.iface()
}

//
// Statuses of a weak ref
//

type wrStatus uint8

// A WeakRef can be in three states: "alive", "dead" or "resurrectable".
//
// To start with it is:
//     alive.
//
// When its value becomes unreachable and the Go GC runs its finalizer it
// changes as follows.
//     alive, dead -> dead
//     resurrectable -> alive
//
// When something gets its value it changes as follows:
//     resurrectable, alive -> resurrectable
//     dead -> dead
// In the last case the returned value is nil.
const (
	wrAlive wrStatus = iota
	wrDead
	wrResurrected
)

//
// Non-retaining reference to an interface value
//

// wiface is an unsafe copy of an interface.  It remembers the type and data of
// a Go interface value, but does not keep it alive.
type wiface [2]uintptr

func getwiface(iface interface{}) wiface {
	return *(*[2]uintptr)(unsafe.Pointer(&iface))
}

func (w wiface) id() uintptr {
	// This is the address containing the interface data.
	return w[1]
}

func (w wiface) iface() interface{} {
	return *(*interface{})(unsafe.Pointer(&w))
}

//
// Values need to be sorted by reverse mark order.  The data structures below help with that.
//
type orderedVal struct {
	val   interface{}
	order int
}

type sortabelVals []orderedVal

var _ sort.Interface = sortabelVals(nil)

func (vs sortabelVals) Len() int {
	return len(vs)
}

func (vs sortabelVals) Less(i, j int) bool {
	return vs[i].order > vs[j].order
}

func (vs sortabelVals) Swap(i, j int) {
	vs[i], vs[j] = vs[j], vs[i]
}

func (vs sortabelVals) vals() []interface{} {
	vals := make([]interface{}, len(vs))
	for i, v := range vs {
		vals[i] = v.val
	}
	return vals
}