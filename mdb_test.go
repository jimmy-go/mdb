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
		err = Close("x")
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
	// already inited
	//	{
	//		expected := "session already done"
	//		di := &mgo.DialInfo{
	//			Addrs:    []string{},
	//			Database: "test",
	//			Timeout:  time.Millisecond,
	//		}
	//	_ = New("x", di, 1, 1)
	//		err = New("x", di, 1, 1)
	//		if err.Error() != expected {
	//			log.Printf("err [%s]", err)
	//			t.Fail()
	//		}
	//	}

	// WORKING SCENARIO
	// execute
	{
		c, ip, port, err := dockertest.SetupMongoContainer()
		if err != nil {
			log.Printf("err [%s]", err)
			t.Fail()
		}
		log.Printf("c [%v]", c)
		log.Printf("ip [%v]", ip)
		log.Printf("port [%v]", port)

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

		// Clean up image.
		c.KillRemove()
	}
}
