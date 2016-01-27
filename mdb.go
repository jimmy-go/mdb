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
	sessions         = make(map[string]*W)
	errNotFound      = errors.New("session not found")
	errAlreadyInited = errors.New("session already done")
	errc             = make(chan error)
)

// W struct define a worker struct.
type W struct {
	Db         string
	Queue      chan jobq.Job
	dispatcher *jobq.Dispatcher
	Workers    []*mgo.Session
	done       chan struct{}
	last       int
	sync.RWMutex
}

// Close close all workers sessions for this prefix.
func (w *W) run(max int) {
	q := make(chan jobq.Job, max)
	w.dispatcher = jobq.NewDispatcher(len(w.Workers), q, errc)
	w.dispatcher.Run()
	go func() {
		select {
		case <-w.done:
			for i := range w.Workers {
				w.Workers[i].Close()
			}
		}
	}()
}

func (w *W) execute(col string, fn func(*mgo.Collection) error) error {
	errc := make(chan error, 1)
	go func() {
		task := func() error {
			w.RLock()
			w.last++
			if w.last >= len(w.Workers) {
				w.last = 0
			}
			w.RUnlock()
			sess := w.Workers[w.last]
			err := fn(sess.DB(w.Db).C(col))
			if err != nil {
				log.Printf("execute err [%s]", err)
			}
			go func() {
				errc <- err
			}()
			return err
		}
		w.dispatcher.Queue <- task
		// return error
	}()
	select {
	case err := <-errc:
		return err
	}
	return nil
}

// New dial a mongodb database and stores the session in a map with a prefix
// so must be called once, then you should be cloning or copying sessions.
//
// It receives a max number of workers that limit the amount of trafic
// and load for the driver. The number must be defined by application
// performance and measurement.
func New(prefix string, options *mgo.DialInfo, maxSessions int) error {
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
	w := &W{
		Db: options.Database,
	}
	for i := 0; i < maxSessions; i++ {
		w.Workers = append(w.Workers, sess.Copy())
	}

	w.run(maxSessions)

	// function must be called in main, mutex is not required.
	sessions[prefix] = w
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
