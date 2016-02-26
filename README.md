#### mdb

MongoDB management for production environments controlled by workers.

[![License MIT](https://img.shields.io/npm/l/express.svg)](http://opensource.org/licenses/MIT)
[![Build Status](https://travis-ci.org/jimmy-go/jobQ.svg?branch=master)](https://travis-ci.org/jimmy-go/jobQ)
[![Go Report Card](https://goreportcard.com/badge/github.com/jimmy-go/mdb)](https://goreportcard.com/report/github.com/jimmy-go/mdb)
[![GoDoc](http://godoc.org/github.com/jimmy-go/mdb?status.png)](http://godoc.org/github.com/jimmy-go/mdb)

##

##### Installation:

```
go get github.com/jimmy-go/mdb
```

##### Usage:

Declare a new connection at start:
```go
mdb.New("mycustomprefix", dialInfo, workers, queueLength)
```

Then you can run queries like this:
```go
err = mdb.Run("mycustomprefix", "mycollection", func(c *mgo.Collection) error {
    return c.Find(nil).Limit(20).All(&items)
})
```

And when you had have done your stuff close the connections and quit all the workers with:
```go
err = mdb.Close("mycustomprefix")
```

Example:

```go
package main

import (
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
		for i := 0; i < *tasks/2; i++ {
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
		for i := 0; i < *tasks/2; i++ {
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
		for i := 0; i < *tasks/2; i++ {
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
	// time.Sleep(3 * time.Second)
	// panic(errors.New("see goroutines"))
}
```

##### Why

In most projects I've been working on, a thin layer for control database consumption was needed.
Some alternatives was adding a load balancer but it has a money cost so I think this solution it better fits me.
Of course comments and critics are welcome :)

LICENCE:

```
The MIT License (MIT)

Copyright (c) 2016 Angel Del Castillo

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```
