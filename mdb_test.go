package mdb

import (
	"fmt"
	"log"
	"testing"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/ory-am/dockertest.v2"
)

func TestNew(t *testing.T) {
	log.SetFlags(log.Lshortfile)

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
	// prefix case insensitive
	{
		expected := "session already done"
		c, ip, port, err := dockertest.SetupMongoContainer()
		if err != nil {
			log.Printf("err [%s]", err)
			t.Fail()
		}
		di := &mgo.DialInfo{
			Addrs:    []string{fmt.Sprintf("%v:%v", ip, port)},
			Database: "test",
			Timeout:  time.Second,
		}
		err = New("A", di, 1, 1)
		err = New("a", di, 1, 1)
		if err == nil {
			t.Fail()
		}
		if err.Error() != expected {
			log.Printf("err [%s]", err)
			t.Fail()
		}
		c.KillRemove()
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
	// invalid timeout
	{
		expected := "invalid timeout duration"
		di := &mgo.DialInfo{
			Addrs:    []string{},
			Database: "test",
			Timeout:  11 * time.Second,
		}
		err := New("f", di, 1, 1)
		if err.Error() != expected {
			log.Printf("err [%s]", err)
			t.Fail()
		}
	}
	// already inited
	{
		c, ip, port, err := dockertest.SetupMongoContainer()
		if err != nil {
			log.Printf("err [%s]", err)
			t.Fail()
		}

		expected := "session already done"
		di := &mgo.DialInfo{
			Addrs:    []string{fmt.Sprintf("%v:%v", ip, port)},
			Database: "test",
			Timeout:  time.Second,
		}
		_ = New("yy", di, 1, 1)
		err = New("yy", di, 1, 1)
		if err == nil {
			log.Printf("err [%s]", err)
			t.Fail()
		} else {
			if err.Error() != expected {
				log.Printf("err [%s]", err)
				t.Fail()
			}
		}

		c.KillRemove()
	}
	// bad jobq workers size
	{
		c, ip, port, err := dockertest.SetupMongoContainer()
		if err != nil {
			log.Printf("err [%s]", err)
			t.Fail()
		}

		expected := "invalid worker size"
		di := &mgo.DialInfo{
			Addrs:    []string{fmt.Sprintf("%v:%v", ip, port)},
			Database: "test",
			Timeout:  time.Second,
		}
		err = New("x100", di, -1, 1)
		if err.Error() != expected {
			log.Printf("err [%s]", err)
			t.Fail()
		}
		c.KillRemove()
	}
	// bad jobq queue size
	{
		c, ip, port, err := dockertest.SetupMongoContainer()
		if err != nil {
			log.Printf("err [%s]", err)
			t.Fail()
		}

		expected := "invalid queue size"
		di := &mgo.DialInfo{
			Addrs:    []string{fmt.Sprintf("%v:%v", ip, port)},
			Database: "test",
			Timeout:  time.Second,
		}
		err = New("x101", di, 1, -1)
		if err.Error() != expected {
			log.Printf("err [%s]", err)
			t.Fail()
		}
		c.KillRemove()
	}

	// WORKING SCENARIO
	// execute
	{
		c, ip, port, err := dockertest.SetupMongoContainer()
		if err != nil {
			log.Printf("err [%s]", err)
			t.Fail()
		}

		di := &mgo.DialInfo{
			Addrs:    []string{fmt.Sprintf("%v:%v", ip, port)},
			Database: "test",
			Timeout:  time.Second,
		}
		err = New("x", di, 1, 1)
		if err != nil {
			log.Printf("err [%s]", err)
			t.Fail()
		}

		// ok Run
		err = Run("x", "coltest", func(c *mgo.Collection) error {
			x := struct {
				Message string `bson:"msg"`
			}{"hello world"}
			return c.Insert(x)
		})
		if err != nil {
			log.Printf("insert err : [%s]", err)
			t.Fail()
		}

		// ok RunWithDB
		err = RunWithDB("x", func(db *mgo.Database) error {
			x := struct {
				Message string `bson:"msg"`
			}{"hello world"}
			return db.C("ottos").Insert(x)
		})
		if err != nil {
			log.Printf("insert with DB err : [%s]", err)
			t.Fail()
		}

		expected := "session not found"
		// prefix not found
		err = Run("x1", "coltest", func(c *mgo.Collection) error {
			x := struct {
				Message string `bson:"msg"`
			}{"hello world"}
			return c.Insert(x)
		})
		if err.Error() != expected {
			log.Printf("insert err : [%s]", err)
			t.Fail()
		}
		err = RunWithDB("x1", func(db *mgo.Database) error {
			x := struct {
				Message string `bson:"msg"`
			}{"hello world"}
			return db.C("ottos").Insert(x)
		})
		if err.Error() != expected {
			log.Printf("insert with DB err : [%s]", err)
			t.Fail()
		}

		expected = "operation timeout"
		// timeout Run
		err = Run("x", "coltest", func(c *mgo.Collection) error {
			time.Sleep(2 * time.Second)
			x := struct {
				Message string `bson:"msg"`
			}{"hello world"}
			return c.Insert(x)
		})
		if err.Error() != expected {
			log.Printf("insert err : [%s]", err)
			t.Fail()
		}

		// timeout RunWithDB
		err = RunWithDB("x", func(db *mgo.Database) error {
			time.Sleep(2 * time.Second)
			x := struct {
				Message string `bson:"msg"`
			}{"hello world"}
			return db.C("ottos").Insert(x)
		})
		if err.Error() != expected {
			log.Printf("insert with DB err : [%s]", err)
			t.Fail()
		}

		// close prefix not found
		expected = "prefix not found"
		err = Close("x3")
		if err.Error() != expected {
			log.Printf("Close x3 err : [%s]", err)
			t.Fail()
		}

		// end work and close
		err = Close("x")
		if err != nil {
			log.Printf("Close x err : [%s]", err)
			t.Fail()
		}

		c.KillRemove()
	}
}
