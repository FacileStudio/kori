package ide

import "sync/atomic"

// A process publishes at most one IDE surface, so the things a session learns
// after its socket is up — that it is publishing at all, and where its
// transcript went — can be said without holding the server anywhere.
var current atomic.Pointer[Server]

// SetSession records the transcript path this process's running session
// writes, so the discovery file and every hello name it. A process that
// publishes nothing ignores the call, which is what keeps its call site free
// of a test.
func SetSession(path string) error {
	return surface().setSession(path)
}

// Close takes this process's IDE surface down: every editor is dropped and the
// socket and the discovery file are removed. A process that never published
// closes nothing, and closing twice is harmless.
func Close() error {
	return surface().Close()
}

// surface is this process's IDE surface, or nil when it publishes none.
func surface() *Server {
	return current.Load()
}

func setCurrent(s *Server) {
	current.Store(s)
}

func clearCurrent(s *Server) {
	current.CompareAndSwap(s, nil)
}
