// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package runtime

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

const (
	generatedSpoolFileName   = "generated_domains.spool.sqlite3"
	generatedBatchSize       = 200
	generatedInsertBatchSize = 1000
)

type spooledGeneratedDomain struct {
	id   int64
	item GeneratedDomain
}

type generatedDomainSpool struct {
	db        *sql.DB
	batchSize int
	mu        sync.Mutex
}

func newGeneratedDomainSpool(generatedOutput string) (*generatedDomainSpool, error) {
	dbPath := generatedSpoolPath(generatedOutput)
	if err := ensureSpoolDir(dbPath); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open generated spool db: %w", err)
	}

	spool := &generatedDomainSpool{db: db, batchSize: generatedBatchSize}
	if err := spool.prepare(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return spool, nil
}

func generatedSpoolPath(generatedOutput string) string {
	clean := strings.TrimSpace(generatedOutput)
	dir := filepath.Dir(clean)
	if dir == "" || dir == "." {
		dir = "."
	}
	return filepath.Join(dir, generatedSpoolFileName)
}

func ensureSpoolDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create generated spool directory: %w", err)
	}
	return nil
}

func (s *generatedDomainSpool) prepare() error {
	if s == nil || s.db == nil {
		return errors.New("generated spool db is not initialized")
	}
	s.db.SetMaxOpenConns(1)
	s.db.SetMaxIdleConns(1)
	return initGeneratedSpoolSchema(s.db)
}

func initGeneratedSpoolSchema(db *sql.DB) error {
	ddl := `
CREATE TABLE IF NOT EXISTS generated_domains (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  domain TEXT NOT NULL,
  risk_score REAL NOT NULL,
  confidence TEXT NOT NULL,
  generated_by TEXT NOT NULL,
  queued INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_generated_domains_queued_id
ON generated_domains(queued, id);
CREATE INDEX IF NOT EXISTS idx_generated_domains_domain
ON generated_domains(domain);
`
	if _, err := db.Exec(ddl); err != nil {
		return fmt.Errorf("create generated spool schema: %w", err)
	}
	return nil
}

func (s *generatedDomainSpool) Close() error {
	if s == nil {
		return nil
	}
	return closeSpoolDB(s.db)
}

func closeSpoolDB(db *sql.DB) error {
	if db == nil {
		return nil
	}
	return db.Close()
}
