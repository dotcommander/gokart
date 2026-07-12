package postgres

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestConfigDSNPrecedenceAndEscaping(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{"url", Config{URL: "postgres://url/db", ConnectionString: "postgres://legacy/db"}, "postgres://url/db"},
		{"legacy", Config{ConnectionString: "postgres://legacy/db"}, "postgres://legacy/db"},
		{"fields", Config{Host: "localhost", Port: 5432, User: "user:name", Password: "p@ss/word?#", DBName: "tenant/db", SSLMode: "verify-full"}, "postgres://user%3Aname:p%40ss%2Fword%3F%23@localhost:5432/tenant%2Fdb?sslmode=verify-full"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.cfg.DSN(); got != test.want {
				t.Fatalf("DSN() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestApplyPoolConfigUsesDefaultsAndOverrides(t *testing.T) {
	poolCfg, err := pgxpool.ParseConfig("postgres://localhost/db")
	if err != nil {
		t.Fatal(err)
	}
	applyPoolConfig(poolCfg, Config{})
	if poolCfg.MaxConns != 25 || poolCfg.MinConns != 5 || poolCfg.MaxConnLifetime != time.Hour || poolCfg.MaxConnIdleTime != 30*time.Minute || poolCfg.HealthCheckPeriod != time.Minute {
		t.Fatalf("default pool config = %+v", poolCfg)
	}

	applyPoolConfig(poolCfg, Config{MaxConns: 8, MinConns: 2, MaxConnLifetime: 2 * time.Hour, MaxConnIdleTime: 10 * time.Minute, HealthCheckPeriod: 15 * time.Second})
	if poolCfg.MaxConns != 8 || poolCfg.MinConns != 2 || poolCfg.MaxConnLifetime != 2*time.Hour || poolCfg.MaxConnIdleTime != 10*time.Minute || poolCfg.HealthCheckPeriod != 15*time.Second {
		t.Fatalf("custom pool config = %+v", poolCfg)
	}
}
