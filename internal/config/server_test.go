package config

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewEmpty - Тестирование создания пустого конфига
func TestNewEmpty(t *testing.T) {
	cfg := NewEmpty()
	d := &Server{
		BaseURL:    DefaultBaseURL,
		NetAddress: DefaultNetAddress,
		TokenExp:   DefaultTokenExp,
	}

	assert.Equal(t, d, cfg, "Несоответствие ожидаемого и получаемого значения")
}

// TestNewParsed - Тестирование singleton версии конфига
func TestNewParsed(t *testing.T) {
	cfg, err := NewParsed()

	// def - Конфиг по умолчанию
	def := NewEmpty()

	// В тесте не заданы консольные флаги, при первом запуске должна прийти ошибка + дефольный конфиг
	assert.Error(t, err)
	assert.Equal(t, def, cfg)

	// Меняем NetAddress у singleton и эталонной версии конфига
	serverCfg.NetAddress = "localhost:2222"
	def.NetAddress = "localhost:2222"

	// Повторный вызов должен вернуть исправленную singleton версию без ошибок
	cfg, err = NewParsed()
	assert.NoError(t, err)
	assert.Equal(t, cfg, def)
}

// TestParse Тестирование парсинга конфига
func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		env     map[string]string
		success bool
		want    *Server
	}{
		{
			name:    "positive - Пустой конфиг",
			args:    make([]string, 0),
			env:     make(map[string]string),
			success: true,
			want: &Server{
				BaseURL:    DefaultBaseURL,
				NetAddress: DefaultNetAddress,
				TokenExp:   DefaultTokenExp,
			},
		},
		{
			name: "positive - Только env",
			args: make([]string, 0),
			env: map[string]string{
				EnvBaseURL:    "http://localhost:1111",
				EnvNetAddress: "localhost:1111",
				EnvFilePath:   "test.bd",
				EnvDSN:        "postgres://postgres:secret@localhost:5432/mydb",
				EnvTokenExp:   fmt.Sprint(time.Hour.Nanoseconds()),
				EnvSecretKey:  "secret",
				EnvAuditFile:  "audit.json",
				EnvAuditURL:   "http://localhost:1111/audit",
			},
			success: true,
			want: &Server{
				BaseURL:    "http://localhost:1111",
				NetAddress: "localhost:1111",
				FilePath:   "test.bd",
				DSN:        "postgres://postgres:secret@localhost:5432/mydb",
				TokenExp:   time.Hour,
				SecretKey:  "secret",
				AuditFile:  "audit.json",
				AuditURL:   "http://localhost:1111/audit",
			},
		},
		{
			name: "positive - Только флаги",
			args: []string{
				"-" + FlagBaseURL, "http://localhost:1111",
				"-" + FlagNetAddress, "localhost:1111",
				"-" + FlagFilePath, "test.bd",
				"-" + FlagDSN, "postgres://postgres:secret@localhost:5432/mydb",
				"-" + FlagTokenExp, fmt.Sprint(time.Hour),
				"-" + FlagSecretKey, "secret",
				"-" + FlagAuditFile, "audit.json",
				"-" + FlagAuditURL, "http://localhost:1111/audit",
			},
			env:     make(map[string]string),
			success: true,
			want: &Server{
				BaseURL:    "http://localhost:1111",
				NetAddress: "localhost:1111",
				FilePath:   "test.bd",
				DSN:        "postgres://postgres:secret@localhost:5432/mydb",
				TokenExp:   time.Hour,
				SecretKey:  "secret",
				AuditFile:  "audit.json",
				AuditURL:   "http://localhost:1111/audit",
			},
		},
		{
			name: "positive - флаги поверх env",
			args: []string{
				"-" + FlagNetAddress, "http://localhost:1111",
				"-" + FlagBaseURL, "localhost:1111",
				"-" + FlagFilePath, "test.bd",
				"-" + FlagDSN, "postgres://postgres:secret@localhost:5432/mydb",
				"-" + FlagTokenExp, fmt.Sprint(time.Hour),
				"-" + FlagSecretKey, "secret",
				"-" + FlagAuditFile, "audit.json",
				"-" + FlagAuditURL, "http://localhost:1111/audit",
			},
			env: map[string]string{
				EnvBaseURL:    "http://localhost:2222",
				EnvNetAddress: "localhost:2222",
				EnvFilePath:   "test2.bd",
				EnvDSN:        "postgres://postgres:secret@localhost:5432/mydb2",
				EnvTokenExp:   fmt.Sprint(time.Minute.Nanoseconds()),
				EnvSecretKey:  "secret2",
				EnvAuditFile:  "audit2.json",
				EnvAuditURL:   "http://localhost:2222/audit",
			},
			success: true,
			want: &Server{
				BaseURL:    "http://localhost:2222",
				NetAddress: "localhost:2222",
				FilePath:   "test2.bd",
				DSN:        "postgres://postgres:secret@localhost:5432/mydb2",
				TokenExp:   time.Minute,
				SecretKey:  "secret2",
				AuditFile:  "audit2.json",
				AuditURL:   "http://localhost:2222/audit",
			},
		},
		{
			name: "negative - Ошибка при парсинге флагов",
			args: []string{
				"-" + FlagNetAddress,
				"-" + FlagBaseURL,
				"-" + FlagFilePath,
				"-" + FlagDSN,
				"-" + FlagTokenExp,
				"-" + FlagSecretKey,
				"-" + FlagAuditFile,
				"-" + FlagAuditURL,
			},
			env:     make(map[string]string),
			success: false,
			want:    nil,
		},
		{
			name: "negative - Некорректное время жизни токена (args)",
			args: []string{
				"-" + FlagTokenExp, "invalid",
			},
			env:     make(map[string]string),
			success: false,
			want:    nil,
		},
		{
			name: "negative - Некорректное время жизни токена (env)",
			args: []string{},
			env: map[string]string{
				EnvTokenExp: "invalid",
			},
			success: false,
			want:    nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// lookupFunc заглушка вместо os.LookupEnv
			lookupFunc := func(name string) (string, bool) {
				value, ok := test.env[name]

				return value, ok
			}

			cfg := NewEmpty()

			// Парсим конфиг
			err := cfg.Parse(test.args, lookupFunc)

			// Если тест должен завершиться успехом - проверяем отсутствие ошибки и сравниваем результаты
			// Иначе проверяем наличие ошибки
			if test.success {
				assert.NoError(t, err, "Ожидалось отсутствие ошибки, но вернулось %s", err)
				assert.Equal(t, test.want, cfg, "Несоответствие результатов ожидаемому значению")
			} else {
				assert.Error(t, err, "Ожидалась ошибка, но получили nil")
			}

		})
	}
}

func TestServer_HasAudit(t *testing.T) {
	cfg := NewEmpty()

	assert.False(t, cfg.HasAudit(), "По умолчанию HasAudit должен быть false")

	cfg.AuditFile = "test.json"

	assert.True(t, cfg.HasAudit(), "При AuditFile=test.json HasAudit должен быть true")

	cfg.AuditURL = "http://localhost:2222"

	assert.True(t, cfg.HasAudit(), "При AuditFile=test.json и cfg.AuditURL=http://localhost:2222 HasAudit должен быть true")

	cfg.AuditFile = ""

	assert.True(t, cfg.HasAudit(), "При cfg.AuditURL=http://localhost:2222 HasAudit должен быть true")
}
