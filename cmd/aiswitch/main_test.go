package main

import (
	"os"
	"testing"
)

func TestMainEntrypoint(t *testing.T) {
	oldArgs, oldExit := os.Args, exit
	t.Cleanup(func() { os.Args, exit = oldArgs, oldExit })
	os.Args = []string{"aiswitch", "version"}
	called, code := false, -1
	exit = func(got int) { called, code = true, got }
	main()
	if !called || code != 0 {
		t.Fatalf("exit called=%v code=%d", called, code)
	}
}
