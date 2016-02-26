package mdb

import (
	"log"
	"testing"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/ory-am/dockertest.v2"
)

func TestNew(t *testing.T) {
	// COMMON ERRORS SCENARIO
	// prefix session empty
	{
		expected := "prefix value is empty"
		err := New("", nil, -1, -1)
		if err == nil {
			t.Fail()
		}
		if err.Error() != expected {
			log.Printf("err [%s]", err)
			t.Fail()
		}
	}
	// dial info is nil
	{
		expected := "dial info is empty"
		err := New("x", nil, -1, -1)
		if err == nil {
			t.Fail()
		}
		if err.Error() != expected {
			log.Printf("err [%s]", err)
			t.Fail()
		}
	}
	// bad workers size count.
	{
		expected := "invalid workers size count"
		di := &mgo.DialInfo{}
		err := New("x", di, -1, -1)
		if err == nil {
			t.Fail()
		}
		if err.Error() != expected {
			log.Printf("err [%s]", err)
			t.Fail()
		}
	}
	// bad queue length.
	{
		expected := "invalid queue length, must be greater than 0"
		di := &mgo.DialInfo{}
		err := New("x", di, 1, -1)
		if err == nil {
			t.Fail()
		}
		if err.Error() != expected {
			log.Printf("err [%s]", err)
			t.Fail()
		}
		Close("x")
	}
	// fail connection.
	{
		expected := "no reachable servers"
		di := &mgo.DialInfo{
			Addrs:    []string{"localhost:27017"},
			Database: "test",
			Timeout:  time.Millisecond,
		}
		err := New("x", di, 1, 1)
		if err.Error() != expected {
			log.Printf("err [%s]", err)
			t.Fail()
		}
	}

	// WORKING SCENARIO
	// execute
	{
		var db *mgo.Session
		c, err := dockertest.ConnectToMongoDB(15, time.Second, func(url string) bool {
			// Check if postgres is responsive...
			var err error
			db, err = mgo.Dial(url)
			if err != nil {
				return false
			}
			return db.Ping() == nil
		})
		if err != nil {
			log.Fatalf("Could not connect to database: %s", err)
			t.Fail()
		}

		// Close database connection.
		db.Close()

		di := &mgo.DialInfo{
			Addrs:    []string{"localhost:27017"},
			Database: "test",
			Timeout:  time.Second,
		}
		err = New("x", di, 1, 1)
		if err != nil {
			log.Printf("err [%s]", err)
			t.Fail()
		}

		// Clean up image.
		c.KillRemove()
	}
}
