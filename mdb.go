package mdb

import (
	"errors"
	"log"
	"sync"

	"gopkg.in/mgo.v2"

	"github.com/jimmy-go/jobq"
)

var (
	// sessions store all sessions.
	sessions         = make(map[string]*Worker)
	errNotFound      = errors.New("session not found")
	errAlreadyInited = errors.New("session already done")
)

// Worker struct define a worker struct.
type Worker struct {
	Db         string
	queue      chan jobq.Job
	dispatcher *jobq.Dispatcher
	Sessions   []*mgo.Session
	done       chan struct{}
	last       int
	sync.RWMutex
}

// New dial a mongodb database and stores the session in a map with a prefix
// so must be called once, then you should be cloning or copying sessions.
//
// It receives a max number of workers that limit the amount of trafic
// and load for the driver. The number must be defined by application
// performance and measurement.
func New(prefix string, options *mgo.DialInfo, workers, queueLength int) error {
	if options == nil {
		return errors.New("dial info is required")
	}
	_, ok := sessions[prefix]
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
		Db:    options.Database,
		queue: make(chan jobq.Job, queueLength),
	}
	for i := 0; i < workers; i++ {
		w.Sessions = append(w.Sessions, sess.Copy())
	}

	w.run(workers)

	// function must be called in main, mutex is not required.
	sessions[prefix] = w
	return nil
}

// Close close all workers sessions for this prefix.
func (w *Worker) run(maxWorkers int) {
	errc := make(chan error)
	go func() {
		for err := range errc {
			if err != nil {
				log.Printf("Worker : run : err [%s]", err)
			}
		}
	}()
	w.dispatcher = jobq.NewDispatcher(maxWorkers, w.queue, errc)
	w.dispatcher.Run()
	select {
	case <-w.done:
		log.Printf("run : <-w.done")
		for i := range w.Sessions {
			w.Sessions[i].Close()
		}
	default:
	}
}

func (w *Worker) execute(col string, fn func(*mgo.Collection) error) error {
	errc := make(chan error, 1)
	task := func() error {
		w.RLock()
		w.last++
		if w.last >= len(w.Sessions) {
			w.last = 0
		}
		w.RUnlock()
		sess := w.Sessions[w.last]
		errc <- fn(sess.DB(w.Db).C(col))
		return nil
	}
	select {
	case w.queue <- task:
	}
	select {
	case err := <-errc:
		return err
	}
	return nil
}

// Run pass a query to the job queue. If queue is full then must be wait until
// some worker is empty.
func Run(prefix, col string, fn func(*mgo.Collection) error) error {
	w, ok := sessions[prefix]
	if !ok {
		return errNotFound
	}
	return w.execute(col, fn)
}
