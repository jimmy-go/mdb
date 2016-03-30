package mdb

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/jimmy-go/jobq"
	"gopkg.in/mgo.v2"
)

var (
	// teams store all sessions.
	teams = make(map[string]*Worker)

	errPrefixEmpty    = errors.New("prefix value is empty")
	errNotFound       = errors.New("session not found")
	errAlreadyInited  = errors.New("session already done")
	errDialInfoEmpty  = errors.New("dial info is empty")
	errTimeout        = errors.New("operation timeout")
	errInvalidTimeout = errors.New("invalid timeout duration")
	errPrefixNotFound = errors.New("prefix not found")
)

// Worker struct define a worker struct.
type Worker struct {
	Db         string
	dispatcher *jobq.Dispatcher
	sessionc   chan *mgo.Session
	done       chan struct{}
	timeout    time.Duration
	size       int
	last       int32
	sync.RWMutex
}

// New dial a mongodb database and stores the session in a map with a prefix
// so must be called once, then you should be cloning or copying sessions.
//
// It receives a max number of workers that limit the amount of traffic
// and load for the driver. The number must be defined by application
// performance and measurement.
func New(prefix string, options *mgo.DialInfo, workers, Qlen int) error {
	if len(prefix) < 1 {
		return errPrefixEmpty
	}
	prefix = strings.ToLower(prefix)
	if options == nil {
		return errDialInfoEmpty
	}
	if options.Timeout > time.Duration(10*time.Second) {
		return errInvalidTimeout
	}
	_, ok := teams[prefix]
	if ok {
		return errAlreadyInited
	}
	// connect logic.
	sess, err := mgo.DialWithInfo(options)
	if err != nil {
		return err
	}
	// TODO; Let user define SetMode.
	sess.SetMode(mgo.Strong, true)

	w := &Worker{
		Db:      options.Database,
		timeout: options.Timeout,
		size:    Qlen,
	}

	w.dispatcher, err = jobq.New(workers, Qlen)
	if err != nil {
		return err
	}

	w.sessionc = make(chan *mgo.Session, workers)
	for i := 0; i < workers; i++ {
		w.sessionc <- sess.Copy()
	}

	teams[prefix] = w
	return nil
}

// stop close all workers sessions for this prefix.
func (w *Worker) stop() {
	for sess := range w.sessionc {
		sess.Close()
		// if no more sessions available then return.
		if len(w.sessionc) < 1 {
			return
		}
	}
}

func (w *Worker) execute(col string, fn func(*mgo.Collection) error) error {
	errc := make(chan error, 1)
	task := func() error {
		// take session from worker.
		select {
		case session := <-w.sessionc:
			err := fn(session.DB(w.Db).C(col))
			select {
			case errc <- err:
			}

			// return session to worker.
			select {
			case w.sessionc <- session:
			}
		}
		return nil
	}
	w.dispatcher.Add(task)
	select {
	case err := <-errc:
		return err
	case <-time.After(w.timeout):
		return errTimeout
	}
}

func (w *Worker) executeDb(fn func(*mgo.Database) error) error {
	errc := make(chan error, 1)
	task := func() error {
		// take session from worker.
		select {
		case session := <-w.sessionc:
			err := fn(session.DB(w.Db))
			select {
			case errc <- err:
			}

			// return session to worker.
			select {
			case w.sessionc <- session:
			}
		}
		return nil
	}
	w.dispatcher.Add(task)
	select {
	case err := <-errc:
		return err
	case <-time.After(w.timeout):
		return errTimeout
	}
}

// Run pass a query to the job queue. If queue is full then must be wait until
// some worker is empty.
func Run(prefix, col string, fn func(*mgo.Collection) error) error {
	w, ok := teams[prefix]
	if !ok {
		return errNotFound
	}
	return w.execute(col, fn)
}

// RunWithDB is like Run for specific cases where pass a mgo.Database is required.
func RunWithDB(prefix string, fn func(db *mgo.Database) error) error {
	w, ok := teams[prefix]
	if !ok {
		return errNotFound
	}
	return w.executeDb(fn)
}

// Close closes all worker sessions from prefix session.
func Close(prefix string) error {
	w, ok := teams[prefix]
	if !ok {
		return errPrefixNotFound
	}
	w.stop()
	return nil
}
