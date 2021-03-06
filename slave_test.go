package turtleDB

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestSlave(t *testing.T) {
	var (
		master *Turtle
		slave  *Slave
		err    error
	)

	if master, err = New("testing", ".test_master", testFM); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(".test_master")
	defer master.Close()

	if slave, err = NewSlave("testing", ".test_slave", testFM, &testImporter{master}, 1); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(".test_slave")
	defer slave.Close()

	slave.SetVerbosity(VerbosityError | VerbosityNotification | VerbositySuccess)

	if err = master.Update(func(txn Txn) (err error) {
		var bkt Bucket
		if bkt, err = txn.Create("bkt"); err != nil {
			return
		}

		if err = bkt.Put("1", testVal1); err != nil {
			return
		}

		if err = bkt.Put("2", testVal2); err != nil {
			return
		}

		if err = bkt.Put("3", testVal3); err != nil {
			return
		}

		return
	}); err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second)

	if err = slave.Read(func(txn Txn) (err error) {
		var bkt Bucket
		if bkt, err = txn.Get("bkt"); err != nil {
			return
		}

		if err = testValueBytes(bkt, "1", testVal1); err != nil {
			return
		}

		if err = testValueBytes(bkt, "2", testVal2); err != nil {
			return
		}

		if err = testValueBytes(bkt, "3", testVal3); err != nil {
			return
		}

		return
	}); err != nil {
		t.Fatal(err)
	}

	if err = master.Update(func(txn Txn) (err error) {
		var bkt Bucket
		if bkt, err = txn.Get("bkt"); err != nil {
			return
		}

		if err = bkt.Delete("2"); err != nil {
			return
		}

		return
	}); err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second)

	if err = slave.Read(func(txn Txn) (err error) {
		var bkt Bucket
		if bkt, err = txn.Get("bkt"); err != nil {
			return
		}

		if err = testValueBytes(bkt, "1", testVal1); err != nil {
			return
		}

		// Key of "2" should not exist anymore
		if err = testValueBytes(bkt, "2", testVal2); err != ErrKeyDoesNotExist {
			return fmt.Errorf("Expected an error of \"%s\" and received \"%v\"", ErrKeyDoesNotExist, err)
		}

		if err = testValueBytes(bkt, "3", testVal3); err != nil {
			return
		}

		return
	}); err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second * 3)
}

func testValueBytes(bkt Bucket, key string, b []byte) (err error) {
	var (
		val  Value
		valB []byte
		ok   bool
	)

	if val, err = bkt.Get(key); err != nil {
		return
	}

	if valB, ok = val.([]byte); !ok {
		return ErrInvalidType
	}

	if strA, strB := string(b), string(valB); strA != strB {
		return fmt.Errorf("invalid value, expected \"%s\" and received  \"%s\"", strA, strB)
	}

	return
}

type testImporter struct {
	master *Turtle
}

func (t *testImporter) Import(txnID string) (rc io.ReadCloser, err error) {
	buf := bytes.NewBuffer(nil)
	if err = t.master.Export(txnID, buf); err != nil {
		return
	}

	rc = ioutil.NopCloser(buf)
	return
}
