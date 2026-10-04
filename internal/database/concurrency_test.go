package database

import (
	"context"
	"database/sql"
	"sync"
	"testing"
)

func TestForeignKeysAreEnabledOnEveryConnection(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	connections := []*sql.Conn{}
	defer func() {
		for _, conn := range connections {
			conn.Close()
		}
	}()
	for i := 0; i < 4; i++ {
		conn, err := db.Conn(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		connections = append(connections, conn)
		var enabled int
		if err = conn.QueryRowContext(context.Background(), `PRAGMA foreign_keys`).Scan(&enabled); err != nil || enabled != 1 {
			t.Fatalf("foreign keys disabled on connection %d: %v", i, err)
		}
	}
}

func TestConcurrentPageReadsAndMemoryWrites(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := RecordStore{DB: db}
	var group sync.WaitGroup
	for i := 0; i < 12; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for n := 0; n < 15; n++ {
				if _, err := store.Create("activity_log", map[string]any{"action": "memory saved"}); err != nil {
					t.Error(err)
					return
				}
				if _, err := store.List("activity_log", "", 20); err != nil {
					t.Error(err)
					return
				}
				if _, err := db.ListProjects(); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}
	group.Wait()
}
