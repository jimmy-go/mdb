package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"runtime"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/jimmy-go/mdb"
)

var (
	workersCount = flag.Int("workers", 3, "Number of workers.")
	qLen         = flag.Int("queue", 10, "Number of queue works.")
	tasks        = flag.Int("tasks", 20, "Number of tasks to run in this example.")

	hosts  = flag.String("host", "", "Mongo host.")
	port   = flag.Int("port", 27107, "Mongo port.")
	dbName = flag.String("database", "", "Mongo database.")
	u      = flag.String("username", "", "Mongo username.")
	p      = flag.String("password", "", "Mongo password.")
)

const (
	pref = "MONGO"
	col  = "items_test"
)

// Post struct.
type Post struct {
	ID   bson.ObjectId `bson:"_id"`
	Link string        `bson:"link"`
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	log.SetFlags(log.Lshortfile)
	log.Printf("workers [%d]", *workersCount)
	log.Printf("queue size [%d]", *qLen)
	log.Printf("tasks [%d]", *tasks)

	di := &mgo.DialInfo{
		Addrs:    []string{*hosts + ":" + fmt.Sprintf("%d", *port)},
		Timeout:  1 * time.Second,
		Database: *dbName,
		Username: *u,
		Password: *p,
	}

	// init new session

	// pref: all sessions require a prefix, this make it easy for you to handle
	// multiple connections.
	// di: dial info for MongoDB.
	// workersCount: define how many workers are going to be running queries,
	// each worker has a session copy that will need to be shutdown by calling mdb.Close(prefix)
	// qLen: define the queue length the prefix connection can handle, when is
	// exceeded all left jobs are paused until some job in queue it is dispatched
	// releasing slots.
	err := mdb.New(pref, di, *workersCount, *qLen)
	if err != nil {
		panic(err)
	}
	log.Println("main : New done")

	// make some reads from db.
	go func() {
		for i := 0; i < *tasks; i++ {
			func(ii int) {
				now := time.Now()
				var items []*Post
				// Run queries passing prefix connections, collection name and
				// function caller.
				err := mdb.Run(pref, col, func(c *mgo.Collection) error {
					err := c.Find(nil).Limit(10).All(&items)
					if err != nil {
						log.Printf("main : err [%s]", err)
					}
					return err
				})
				if err != nil {
					log.Printf("main : err [%s]", err)
					return
				}
				log.Printf("main : done find [%s] i [%v] results [%v]", time.Since(now), ii, len(items))
			}(i)
		}
	}()

	// do some writes.
	go func() {
		for i := 0; i < *tasks; i++ {
			func(ii int) {
				now := time.Now()
				// Run queries passing prefix connections, collection name and
				// function caller.
				if err := mdb.Run(pref, col, func(c *mgo.Collection) error {
					return c.Insert(bson.M{"link": fmt.Sprintf("%v", ii)})
				}); err != nil {
					log.Printf("main : err [%s]", err)
					return
				}
				log.Printf("main : done insert [%s] i [%v]", time.Since(now), ii)
			}(i)
		}
	}()

	// run operations with *mgo.Database
	go func() {
		for i := 0; i < *tasks; i++ {
			func(ii int) {
				now := time.Now()
				var items []*Post
				// you also can run queries that need a database struct.
				// But this is discouraged.
				if err := mdb.RunWithDB(pref, func(db *mgo.Database) error {
					return db.C(col).Find(nil).Limit(20).All(&items)
				}); err != nil {
					log.Printf("main : err [%s]", err)
					return
				}
				log.Printf("main : DB done find [%s] i [%v] results [%v]", time.Since(now), ii, len(items))
			}(i)
		}
	}()

	time.Sleep(5 * time.Second)
	// close sessions on exit
	mdb.Close(pref)

	// sometimes I want to know how many goroutines are running.
	time.Sleep(3 * time.Second)
	panic(errors.New("see goroutines"))
}
