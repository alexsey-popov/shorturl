package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestServer_ParseFromFile(t *testing.T) {
	tempDir := t.TempDir()

	validConfigPath := filepath.Join(tempDir, "config.json")
	validJSON := `{
		"base_url": "http://localhost:9090",
		"address": "localhost:9090",
		"store_file": "/tmp/store.json",
		"database_dsn": "postgres://user:pass@localhost:5432/db",
		"token_exp": 7200000000000,
		"secret_key": "topsecret",
		"audit_file": "/tmp/audit.json",
		"audit_url": "http://localhost:9090/audit",
		"enable_https": true
	}`
	err := os.WriteFile(validConfigPath, []byte(validJSON), 0644)
	require.NoError(t, err)

	invalidJSONPath := filepath.Join(tempDir, "invalid.json")
	err = os.WriteFile(invalidJSONPath, []byte(`{invalid_json:`), 0644)
	require.NoError(t, err)

	partialConfigPath := filepath.Join(tempDir, "partial.json")
	partialJSON := `{
		"base_url": "http://localhost:3000",
		"address": "localhost:3000"
	}`
	err = os.WriteFile(partialConfigPath, []byte(partialJSON), 0644)
	require.NoError(t, err)

	tests := []struct {
		name        string
		filePath    string
		initialCfg  *Server
		expectedCfg *Server
		wantErr     bool
	}{
		{
			name:       "positive - валидный файл с полным набором параметров",
			filePath:   validConfigPath,
			initialCfg: NewEmpty(),
			expectedCfg: &Server{
				BaseURL:     "http://localhost:9090",
				NetAddress:  "localhost:9090",
				FilePath:    "/tmp/store.json",
				DSN:         "postgres://user:pass@localhost:5432/db",
				TokenExp:    time.Hour * 2,
				SecretKey:   "topsecret",
				AuditFile:   "/tmp/audit.json",
				AuditURL:    "http://localhost:9090/audit",
				EnableHTTPS: true,
			},
			wantErr: false,
		},
		{
			name:       "positive - валидный файл с частичными параметрами поверх дефолтных",
			filePath:   partialConfigPath,
			initialCfg: NewEmpty(),
			expectedCfg: &Server{
				BaseURL:     "http://localhost:3000",
				NetAddress:  "localhost:3000",
				FilePath:    "",
				DSN:         "",
				TokenExp:    DefaultTokenExp,
				SecretKey:   "",
				AuditFile:   "",
				AuditURL:    "",
				EnableHTTPS: DefaultEnableHTTPS,
			},
			wantErr: false,
		},
		{
			name:        "negative - несуществующий файл",
			filePath:    filepath.Join(tempDir, "non_existent.json"),
			initialCfg:  NewEmpty(),
			expectedCfg: nil,
			wantErr:     true,
		},
		{
			name:        "negative - некорректный JSON",
			filePath:    invalidJSONPath,
			initialCfg:  NewEmpty(),
			expectedCfg: nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.initialCfg
			err := cfg.ParseFromFile(tt.filePath)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCfg, cfg)
			}
		})
	}
}

func TestParse_PriorityOrder(t *testing.T) {
	tempDir := t.TempDir()

	flagConfigPath := filepath.Join(tempDir, "flag_config.json")
	flagJSON := `{
		"address": "from_flag_file:1111",
		"base_url": "http://from_flag_file:1111",
		"store_file": "flag_file.db",
		"secret_key": "flag_key"
	}`
	err := os.WriteFile(flagConfigPath, []byte(flagJSON), 0644)
	require.NoError(t, err)

	envConfigPath := filepath.Join(tempDir, "env_config.json")
	envJSON := `{
		"address": "from_env_file:2222",
		"base_url": "http://from_env_file:2222",
		"store_file": "env_file.db"
	}`
	err = os.WriteFile(envConfigPath, []byte(envJSON), 0644)
	require.NoError(t, err)

	t.Run("иерархия: defaults -> flag config -> env config -> cli flags -> env vars", func(t *testing.T) {
		args := []string{
			"-" + FlagConfigFile, flagConfigPath,
			"-" + FlagNetAddress, "from_cli_flag:3333",
			"-" + FlagBaseURL, "http://from_cli_flag:3333",
		}

		env := map[string]string{
			EnvConfigFile: envConfigPath,
			EnvNetAddress: "from_env_var:4444",
		}

		lookupFunc := func(name string) (string, bool) {
			val, ok := env[name]
			return val, ok
		}

		cfg := NewEmpty()
		err := cfg.Parse(args, lookupFunc)
		require.NoError(t, err)

		// 1. NetAddress: env var (4444) перекрывает CLI flag (3333)
		assert.Equal(t, "from_env_var:4444", cfg.NetAddress)
		// 2. BaseURL: CLI flag (3333) перекрывает env config file (2222)
		assert.Equal(t, "http://from_cli_flag:3333", cfg.BaseURL)
		// 3. FilePath: env config file (env_file.db) перекрывает flag config file (flag_file.db)
		assert.Equal(t, "env_file.db", cfg.FilePath)
		// 4. SecretKey: flag config file (flag_key) перекрывает default ("")
		assert.Equal(t, "flag_key", cfg.SecretKey)
		// 5. Default значения сохраняются, если не переопределены
		assert.Equal(t, DefaultTokenExp, cfg.TokenExp)
		assert.Equal(t, DefaultEnableHTTPS, cfg.EnableHTTPS)
	})

	t.Run("ошибка при некорректном файле из флага -c", func(t *testing.T) {
		args := []string{
			"-" + FlagConfigFile, filepath.Join(tempDir, "non_existent.json"),
		}
		lookupFunc := func(name string) (string, bool) { return "", false }

		cfg := NewEmpty()
		err := cfg.Parse(args, lookupFunc)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ошибка при парсинге конфига из файла(flag)")
	})

	t.Run("ошибка при некорректном файле из env CONFIG", func(t *testing.T) {
		args := []string{}
		lookupFunc := func(name string) (string, bool) {
			if name == EnvConfigFile {
				return filepath.Join(tempDir, "non_existent.json"), true
			}
			return "", false
		}

		cfg := NewEmpty()
		err := cfg.Parse(args, lookupFunc)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ошибка при парсинге конфига из файла(env)")
	})
}
