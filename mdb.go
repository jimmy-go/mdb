package mdb

import (
	"errors"
	"log"
	"sync"
	"time"

	"gopkg.in/mgo.v2"

	"github.com/jimmy-go/jobq"
)

var (
	// teams store all sessions.
	teams = make(map[string]*Worker)

	errNotFound      = errors.New("session not found")
	errAlreadyInited = errors.New("session already done")
	errDialInfoEmpty = errors.New("dial info is empty")
	errTimeout       = errors.New("operation timeout")
)

// Worker struct define a worker struct.
type Worker struct {
	Db         string
	dispatcher *jobq.Dispatcher
	Sessions   []*mgo.Session
	done       chan struct{}
	timeout    time.Duration
	size       int
	last       int32
	sync.RWMutex
}

// New dial a mongodb database and stores the session in a map with a prefix
// so must be called once, then you should be cloning or copying sessions.
//
// It receives a max number of workers that limit the amount of trafic
// and load for the driver. The number must be defined by application
// performance and measurement.
func New(prefix string, options *mgo.DialInfo, workers, Qlen int) error {
	if options == nil {
		return errDialInfoEmpty
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

	if options.Timeout > time.Duration(10*time.Second) {
		log.Printf("New [%s] timeout is too high", prefix)
	}

	w := &Worker{
		Db:   options.Database,
		size: Qlen,
	}
	for i := 0; i < workers; i++ {
		w.Sessions = append(w.Sessions, sess.Copy())
	}

	errc := make(chan error, 1)
	go func() {
		for err := range errc {
			if err != nil {
				log.Printf("New : err [%s]", err)
			}
		}
	}()
	w.dispatcher, err = jobq.New(workers, Qlen, errc)
	if err != nil {
		return err
	}

	teams[prefix] = w
	return nil
}

// stop close all workers sessions for this prefix.
func (w *Worker) stop() {
	w.RLock()
	for i := range w.Sessions {
		w.Sessions[i].Close()
	}
	w.RUnlock()
}

func (w *Worker) session() *mgo.Session {
	w.last++
	if w.last >= int32(len(w.Sessions)) {
		w.last = 0
	}
	w.RLock()
	s := w.Sessions[w.last]
	w.RUnlock()
	return s
}

func (w *Worker) execute(col string, fn func(*mgo.Collection) error) error {
	errc := make(chan error, 1)
	task := func() error {
		sess := w.session()
		errc <- fn(sess.DB(w.Db).C(col))
		return nil
	}
	w.dispatcher.Add(task)
	select {
	case err := <-errc:
		return err
	case <-time.After(w.timeout):
	}
	return nil
}

func (w *Worker) executeDb(col string, fn func(*mgo.Database) error) error {
	errc := make(chan error, 1)
	task := func() error {
		sess := w.session()
		errc <- fn(sess.DB(w.Db))
		return nil
	}
	w.dispatcher.Add(task)
	select {
	case err := <-errc:
		return err
	case <-time.After(w.timeout):
		return errTimeout
	}
	return nil
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
func RunWithDB(prefix, col string, fn func(*mgo.Database) error) error {
	w, ok := teams[prefix]
	if !ok {
		return errNotFound
	}
	return w.executeDb(col, fn)
}

// Close closes all worker sessions from prefix session.
func Close(prefix string) {
	w, ok := teams[prefix]
	if !ok {
		return
	}
	w.stop()
}
