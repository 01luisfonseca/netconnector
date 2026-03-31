package server

import (
	"testing"
)

func TestSQLiteDB(t *testing.T) {
	// Use in-memory SQLite for testing
	db, err := NewSQLiteDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory db: %v", err)
	}
	defer db.Close()

	t.Run("AddAndGet", func(t *testing.T) {
		sub := "test"
		clientID := "client-123"
		if err := db.AddMapping(sub, clientID); err != nil {
			t.Errorf("AddMapping failed: %v", err)
		}

		got, err := db.GetClientIDBySubdomain(sub)
		if err != nil {
			t.Errorf("GetClientIDBySubdomain failed: %v", err)
		}
		if got != clientID {
			t.Errorf("expected clientID %s, got %s", clientID, got)
		}
	})

	t.Run("GetNonExistent", func(t *testing.T) {
		got, err := db.GetClientIDBySubdomain("non-existent")
		if err != nil {
			t.Errorf("GetClientIDBySubdomain failed: %v", err)
		}
		if got != "" {
			t.Errorf("expected empty string for non-existent subdomain, got %s", got)
		}
	})

	t.Run("IsClientIDRegistered", func(t *testing.T) {
		clientID := "reg-client"
		_ = db.AddMapping("sub1", clientID)

		exists, err := db.IsClientIDRegistered(clientID)
		if err != nil {
			t.Errorf("IsClientIDRegistered failed: %v", err)
		}
		if !exists {
			t.Error("expected clientID to be registered")
		}

		exists, _ = db.IsClientIDRegistered("unknown")
		if exists {
			t.Error("expected unknown clientID to not be registered")
		}
	})

	t.Run("RemoveMapping", func(t *testing.T) {
		sub := "to-remove"
		_ = db.AddMapping(sub, "id")

		if err := db.RemoveMapping(sub); err != nil {
			t.Errorf("RemoveMapping failed: %v", err)
		}

		got, _ := db.GetClientIDBySubdomain(sub)
		if got != "" {
			t.Error("expected subdomain to be removed")
		}
	})

	t.Run("ListMappings", func(t *testing.T) {
		_ = db.AddMapping("a", "1")
		_ = db.AddMapping("b", "2")

		list, err := db.ListMappings()
		if err != nil {
			t.Errorf("ListMappings failed: %v", err)
		}

		if list["a"] != "1" || list["b"] != "2" {
			t.Errorf("ListMappings returned incorrect data: %v", list)
		}
	})
}
