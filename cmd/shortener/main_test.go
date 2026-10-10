package main

import (
	"testing"

	"github.com/alexsey-popov/shorturl/internal/repository/indb"
	"github.com/stretchr/testify/assert"
)

func TestConnectDB(t *testing.T) {
	// Некорректный DSN должен приводить к ошибке подключения или миграций
	db, err := indb.ConnectDB("postgres://invalid:invalid@localhost:5432/nonexistent?sslmode=disable")
	assert.Error(t, err)
	if db != nil {
		db.Close()
	}
}

func TestBuildVars(t *testing.T) {
	assert.NotEmpty(t, buildVersion)
	assert.NotEmpty(t, buildDate)
	assert.NotEmpty(t, buildCommit)
}
