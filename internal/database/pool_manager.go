package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/MrVirul/mini-orders-service/internal/tenant"
)

type PoolManager struct {
	mu    sync.RWMutex
	pools map[string]*sql.DB
	open  func(dsn string) (*sql.DB, error)
}

func NewPoolManager() *PoolManager {
	return NewPoolManagerWithOpener(func(dsn string) (*sql.DB, error) {
		return sql.Open("mysql", dsn)
	})
}

// the opener is injected so tests can count how many pools get created
func NewPoolManagerWithOpener(open func(string) (*sql.DB, error)) *PoolManager {
	return &PoolManager{pools: make(map[string]*sql.DB), open: open}
}

func buildDSN(t *tenant.Tenant) string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=UTC&timeout=3s",
		t.Username, t.Password, t.Host, t.Port, t.DatabaseName)
}

func (p *PoolManager) getOrCreate(t *tenant.Tenant) (*sql.DB, error) {
	// 1. fast path: many requests can read at the same time
	p.mu.RLock()
	db, ok := p.pools[t.Code]
	p.mu.RUnlock()
	if ok {
		return db, nil
	}

	// 2. slow path: only one request at a time may create a pool
	p.mu.Lock()
	defer p.mu.Unlock()

	// 3. check again: someone may have created it while we waited for the lock
	if db, ok := p.pools[t.Code]; ok {
		return db, nil
	}

	db, err := p.open(buildDSN(t))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	p.pools[t.Code] = db
	return db, nil
}

func (p *PoolManager) Get(ctx context.Context, t *tenant.Tenant) (*sql.DB, error) {
	db, err := p.getOrCreate(t)
	if err != nil {
		return nil, err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("ping tenant db %s: %w", t.Code, err)
	}
	return db, nil
}

func (p *PoolManager) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for code, db := range p.pools {
		_ = db.Close()
		delete(p.pools, code)
	}
}
