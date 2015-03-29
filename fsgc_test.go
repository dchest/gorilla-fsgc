// Written in 2015 by Dmitry Chestnykh.
//
// To the extent possible under law, the author have dedicated all copyright
// and related and neighboring rights to this software to the public domain
// worldwide. This software is distributed without any warranty.
// http://creativecommons.org/publicdomain/zero/1.0/

package fsgc

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestGC(t *testing.T) {
	dir, err := ioutil.TempDir("", "fsgc")
	if err != nil {
		t.Fatal(err)
	}
	f1 := filepath.Join(dir, "session_1")
	f2 := filepath.Join(dir, "session_2")
	if err := ioutil.WriteFile(f1, []byte("session1"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(f2, []byte("session2"), 0600); err != nil {
		t.Fatal(err)
	}
	// Make f1 expired.
	if err := os.Chtimes(f1, time.Now(), time.Now().Add(-(DefaultMaxAge + 10*time.Minute))); err != nil {
		t.Fatal(err)
	}
	gc := New(dir).Interval(100 * time.Millisecond).Start()
	defer gc.Stop()
	time.Sleep(500 * time.Millisecond)
	runtime.Gosched()
	// Check that f2 exists and f1 doesn't.
	_, err = os.Lstat(f1)
	if err == nil {
		t.Fatalf("fsgc: file %s exist, but should have been removed by GC", f1)
	}
	if !os.IsNotExist(err) {
		// some other error
		t.Fatal(err)
	}
	_, err = os.Lstat(f2)
	if err != nil {
		t.Fatal(err)
	}
	os.RemoveAll(dir)
}
