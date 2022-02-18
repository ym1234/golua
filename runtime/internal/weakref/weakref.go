// Package weakref implements weak refs and weak ref pools to be used by the
// Golua runtime.
//
// Two interfaces WeakRef and Pool are defined and the packages provides two
// implementations of WeakRefPool.  The Golua runtime has a Pool instance that
// it uses to help with finalizing of Lua values and making sure finalizers do
// not run after the runtime has finished.
//
// SafeWeakRefPool is a simple implementation whose strategy is to keep all
// values alive as long as they have live WeakRefs.
//
// UnsafeWeakRefPool makes every effort to let values be GCed when they are only
// reachable via WeakRefs.  It relies on casting interface{} to unsafe pointers
// and back again, which would break if Go were to have a moving GC.
package weakref

// A WeakRef is a weak reference to a value. Its Value() method returns the
// value if it is not dead (meaning that the value is still reachable),
// otherwise it returns nil.
//
// Note that it is valid for a WeakRef to keep its value alive while it is
// alive, although not very efficient.
type WeakRef interface {
	Value() interface{}
}

// A Pool maintains a set of weak references to values.  Its methods are not
// required to be thread-safe insofar as they should not get called
// concurrently.
//
// Each Golua Runtime has a Pool instance to help it manage weak references and
// finalizers.
type Pool interface {

	// Get returns a WeakRef for the given value v.  Calling Get several times
	// with the same value should return the same WeakRef.
	Get(v interface{}) WeakRef

	// Mark indicates that the pool should keep a copy of v when it dies.
	//
	// The associated Golua Runtime marks all values which have a __gc
	// metamethod, so that it can get notified when those values are unreachable
	// via ExtractPendingFinalize, ExtractPendingRelease.
	Mark(v interface{}, flags MarkFlags)

	// ExtractPendingFinalize returns all marked values which are no longer reachable
	// and haven't been returned yet, so that some finalizing code can be run
	// with them.  The returned values are ordered in reverse order of marking
	// (i.e. if v1 was marked before v2, then v2 comes before v1 in the returned
	// list). Further calls should not return the same values again.
	//
	// This is called periodically by the Golua Runtime to run Lua finalizers on
	// GCed values.
	ExtractPendingFinalize() []interface{}

	// ExtractPendingRelease returns all marked values which are no longer
	// reachable, no longer need to be finalized and haven't been returned yet,
	// so that their associated resources can be released.  The returned values
	// are ordered in reverse order of marking (i.e. if v1 was marked before v2,
	// then v2 comes before v1 in the returned list). Further calls should not
	// return the same values again.
	//
	// This is called periodically by the Golua Runtime to release resources
	// associated with GCed values.
	ExtractPendingRelease() []interface{}

	// ExtractAllMarkedFinalize returns all values marked for Lua finalizing,
	// following the same order as ExtractPendingFinalize.  All marked values
	// are cleared in the pool so that they will no longer be returned by this
	// method or ExtractPendingFinalize.
	//
	// Typically this method will be called when the associated Golua Runtime is
	// being closed so that all outstanding Lua finalizers can be called (even
	// if their values might get GCed later).
	ExtractAllMarkedFinalize() []interface{}

	// ExtractAllMarkedRelease returns all values marked for releasing,
	// following the same order as ExtractPendingRelease.  All marked values
	// are cleared in the pool so that they will no longer be returned by this
	// method or ExtractPendingRelease.
	//
	// Typically this method will be called when the associated Golua Runtime is
	// being closed so that all outstanding Lua finalizers can be called (even
	// if their values might get GCed later).
	ExtractAllMarkedRelease() []interface{}
}

type MarkFlags uint8

const (
	Finalize MarkFlags = 1 << iota
	Release
)

// NewPool returns a new WeakRefPool with an appropriate implementation.
func NewPool() Pool {
	return NewUnsafePool()
}
