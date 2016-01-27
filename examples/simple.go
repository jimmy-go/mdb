package main

import (
	"flag"
	"log"
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
	err := mdb.New(pref, di, *maxWs)
	if err != nil {
		panic(err)
	}
	log.Println("main : New done")
	for i := 0; i < *tasks; i++ {
		go func(ii int) {
			now := time.Now()
			var us []*Post
			if err := mdb.Run(pref, "posts", func(c *mgo.Collection) error {
				// Skip is a very expensive operation.
				// for pagination use range queries.
				return c.Find(nil).Limit(6).All(&us)
			}); err != nil {
				log.Printf("main : err [%s]", err)
			} else {
				log.Printf("main : done! [%s] ii [%v] len [%v]", time.Since(now), ii, len(us))
			}
		}(i)
	}
	time.Sleep(1 * time.Minute)
	for {
		log.Println("sleep 1 minute!")
		time.Sleep(1 * time.Minute)
	}
}
