package tests

import (
	"context"
	"database/sql"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/MrVirul/mini-orders-service/internal/database"
	"github.com/MrVirul/mini-orders-service/internal/tenant"
)

func TestGet_ConcurrentRequestsCreateExactlyOnePool(t *testing.T) {
	ctx := context.Background()
	var opened atomic.Int32
	pm := database.NewPoolManagerWithOpener(func(string) (*sql.DB, error) {
		opened.Add(1)
		db, _, err := sqlmock.New()
		return db, err
	})
	defer pm.Close()

	tn := &tenant.Tenant{Code: "pizza_palace", Host: "h", Port: "3306", DatabaseName: "d", Username: "u"}

	const n = 100
	var (
		start sync.WaitGroup
		done  sync.WaitGroup
		mu    sync.Mutex
		got   = map[*sql.DB]struct{}{}
	)
	start.Add(1)
	for i := 0; i < n; i++ {
		done.Add(1)
		go func() {
			defer done.Done()
			start.Wait() // release all goroutines together to maximise contention
			db, err := pm.Get(ctx, tn)
			if err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			got[db] = struct{}{}
			mu.Unlock()
		}()
	}
	start.Done()
	done.Wait()

	if c := opened.Load(); c != 1 {
		t.Fatalf("pools opened = %d, want exactly 1", c)
	}
	if len(got) != 1 {
		t.Fatalf("distinct *sql.DB returned = %d, want 1", len(got))
	}
}

func TestGet_SeparatePoolPerTenant(t *testing.T) {
	ctx := context.Background()
	var opened atomic.Int32
	pm := database.NewPoolManagerWithOpener(func(string) (*sql.DB, error) {
		opened.Add(1)
		db, _, err := sqlmock.New()
		return db, err
	})
	defer pm.Close()

	a, _ := pm.Get(ctx, &tenant.Tenant{Code: "a"})
	b, _ := pm.Get(ctx, &tenant.Tenant{Code: "b"})
	if a == b || opened.Load() != 2 {
		t.Fatalf("want 2 distinct pools, opened=%d", opened.Load())
	}
}
