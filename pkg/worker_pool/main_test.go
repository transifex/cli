package worker_pool

import (
	"testing"
	"time"
)

func TestEmptyPoolCompletes(t *testing.T) {
	pool := New(1, 0, true)
	pool.Start()

	select {
	case <-pool.Wait():
	case <-time.After(time.Second):
		t.Fatal("empty pool did not complete")
	}
}

func TestMakeProgressBarWithNoTasks(t *testing.T) {
	got := makeProgressBar(0, 0)
	want := "[------------------------------] (0 / 0)"
	if got != want {
		t.Fatalf("makeProgressBar(0, 0) = %q, want %q", got, want)
	}
}
