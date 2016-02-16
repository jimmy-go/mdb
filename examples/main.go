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
	maxWs = flag.Int("max-workers", 3, "Number of workers.")
	maxQs = flag.Int("max-queue", 10, "Number of queue works.")
	tasks = flag.Int("tasks", 20, "Number of tasks.")

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
	log.Printf("workers [%d]", *maxWs)
	log.Printf("queue len [%d]", *maxQs)
	log.Printf("tasks [%d]", *tasks)
	log.Printf("hosts [%v]", *hosts)
	log.Printf("port [%v]", *port)
	log.Printf("dbName [%v]", *dbName)
	log.Printf("u [%v]", *u)
	log.Printf("p [%v]", *p)
	log.Printf("collection [%v]", col)

	errc := make(chan error, 1)
	go func() {
		for err := range errc {
			if err != nil {
				log.Printf("main : error channel : err [%s]", err)
			}
		}
	}()

	di := &mgo.DialInfo{
		Addrs:    []string{*hosts + ":" + fmt.Sprintf("%d", *port)},
		Timeout:  1 * time.Second,
		Database: *dbName,
		Username: *u,
		Password: *p,
	}
	err := mdb.New(pref, di, *maxWs, *maxQs)
	if err != nil {
		panic(err)
	}
	log.Println("main : New done")

	// reads
	go func() {
		for i := 0; i < *tasks/2; i++ {
			func(ii int) {
				now := time.Now()
				var items []*Post
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

	// writes
	go func() {
		for i := 0; i < *tasks/2; i++ {
			func(ii int) {
				now := time.Now()
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

	// with
	// db
	go func() {
		for i := 0; i < *tasks/2; i++ {
			func(ii int) {
				now := time.Now()
				var items []*Post
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
	mdb.Close(pref)
	time.Sleep(3 * time.Second)
	panic(errors.New("see goroutines"))
}
