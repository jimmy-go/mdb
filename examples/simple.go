package main

import (
	"errors"
	"flag"
	"log"
	"runtime"
	"time"

	"gopkg.in/mgo.v2"

	"github.com/jimmy-go/mdb"
)

var (
	maxWs = flag.Int("max-workers", 3, "Number of workers.")
	maxQs = flag.Int("max-queue", 10, "Number of queue works.")
	tasks = flag.Int("tasks", 20, "Number of tasks.")

	hosts  = flag.String("host", "", "Mongo host.")
	dbName = flag.String("database", "", "Mongo database.")
	u      = flag.String("username", "", "Mongo username.")
	p      = flag.String("password", "", "Mongo password.")
)

const (
	pref = "MONGO"
)

// Post struct.
type Post struct {
	Link string `bson:"link"`
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	log.SetFlags(log.Lshortfile)
	log.Printf("workers [%d]", *maxWs)
	log.Printf("queue len [%d]", *maxQs)
	log.Printf("tasks [%d]", *tasks)
	log.Printf("hosts [%v]", *hosts)
	log.Printf("dbName [%v]", *dbName)
	log.Printf("u [%v]", *u)
	log.Printf("p [%v]", *p)

	errc := make(chan error)
	go func() {
		for err := range errc {
			if err != nil {
				log.Printf("main : error channel : err [%s]", err)
			}
		}
	}()

	di := &mgo.DialInfo{
		Addrs:    []string{*hosts},
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

	go func() {
		select {
		case <-time.After(30 * time.Second):
			panic(errors.New("panic to see current go rutines"))
		}
	}()

	for i := 0; i < *tasks; i++ {
		func(ii int) {
			now := time.Now()
			var us []Post
			if err := mdb.Run(pref, "posts", func(c *mgo.Collection) error {
				return c.Find(nil).Limit(100).All(&us)
			}); err != nil {
				log.Printf("main : err [%s]", err)
			} else {
				log.Printf("main : done! [%s] job number [%v] results len [%v]", time.Since(now), ii, len(us))
			}
		}(i)
	}

	for {
		select {}
	}
}
