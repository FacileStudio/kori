package sessions

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"time"

	"github.com/FacileStudio/nacelle"
)

// RestoredSession is the outcome of a session restore check at launch.
type RestoredSession struct {
	Conversation []nacelle.Message
	Name         string
	Path         string
	Error        string
}

// RestoreAtLaunch returns the conversation to restore when the client starts:
// the session named by resume (an id or a file path) when one is given, else
// the newest session for root when auto is on.
func RestoreAtLaunch(resume, root string, auto bool) RestoredSession {
	if resume != "" {
		path := ResolveSession(resume)
		if path == "" {
			return RestoredSession{Error: "no session found for \"" + resume + "\""}
		}
		return RestoredSession{Conversation: LoadSession(path), Name: filepath.Base(path), Path: path}
	}
	if auto {
		if root == "" {
			root = "."
		}
		files := ListSessionFiles(root)
		if len(files) > 0 {
			return RestoredSession{Conversation: LoadSession(files[0]), Name: filepath.Base(files[0]), Path: files[0]}
		}
	}
	return RestoredSession{}
}

func cleanup(f *os.File, path string) {
	if f != nil {
		if err := f.Close(); err != nil {
			return
		}
	}
	if err := os.Remove(path); err != nil {
		return
	}
}

func archiveSession(path string) bool {
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	gzPath := path + ".gz"
	if _, err := os.Stat(gzPath); err == nil {
		return false
	}
	gzFile, err := os.OpenFile(gzPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return false
	}
	gz := gzip.NewWriter(gzFile)
	if _, err := gz.Write(content); err != nil {
		cleanup(gzFile, gzPath)
		return false
	}
	if err := gz.Close(); err != nil {
		cleanup(gzFile, gzPath)
		return false
	}
	if err := gzFile.Close(); err != nil {
		cleanup(nil, gzPath)
		return false
	}
	return os.Remove(path) == nil
}

func (l *SessionLog) rotate() {
	if !archiveSession(l.path) {
		l.lastSize = 0
		return
	}
	now := time.Now()
	name := generateSessionID(filepath.Dir(l.path)) + ".jsonl"
	l.path = filepath.Join(filepath.Dir(l.path), name)
	l.lastSize = 0
	l.write(sessionHeader{
		Version: 1,
		Started: now.Format(time.RFC3339Nano),
		Backend: l.backend,
		Model:   l.model,
		Root:    l.root,
		PID:     os.Getpid(),
	})
}
