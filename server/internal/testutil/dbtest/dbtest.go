// Package dbtest gives each test its own freshly migrated Postgres database.
//
// It uses TEST_DATABASE_URL when set (any Postgres the tests may create
// databases on). Otherwise it starts — or reuses — a local Docker container
// named "upstonk-testdb" (postgres:16-alpine on port 55432). It never touches
// DATABASE_URL, so tests can't reach the real (Supabase) database.
// Remove the container with: docker rm -f upstonk-testdb
package dbtest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"upstonk/internal/store"
)

const (
	containerName = "upstonk-testdb"
	image         = "postgres:16-alpine"
	hostPort      = "55432"
	password      = "upstonk-test"
)

var (
	adminOnce sync.Once
	adminURL  string
	adminErr  error
)

// New returns a pool connected to a new, migrated, empty database that is
// dropped when the test ends. It skips the test if no Postgres is available.
func New(t *testing.T) *pgxpool.Pool {
	t.Helper()
	adminOnce.Do(func() { adminURL, adminErr = serverURL() })
	if adminErr != nil {
		t.Skipf("no test Postgres (set TEST_DATABASE_URL or install Docker): %v", adminErr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	name := "upstonk_test_" + randomSuffix()
	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatalf("connect to test server: %v", err)
	}
	defer admin.Close(ctx)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("create test database: %v", err)
	}

	pool, err := store.Open(ctx, withDatabase(adminURL, name))
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
		c, err := pgx.Connect(context.Background(), adminURL)
		if err == nil {
			_, _ = c.Exec(context.Background(), "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)")
			c.Close(context.Background())
		}
	})
	if err := store.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return pool
}

// serverURL returns a URL for a Postgres server the tests may create
// databases on, starting the Docker container if needed.
func serverURL() (string, error) {
	if u := os.Getenv("TEST_DATABASE_URL"); u != "" {
		return u, nil
	}
	if _, err := exec.LookPath("docker"); err != nil {
		return "", fmt.Errorf("docker not found")
	}
	running, _ := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", containerName).Output()
	switch strings.TrimSpace(string(running)) {
	case "true":
	case "false":
		if out, err := exec.Command("docker", "start", containerName).CombinedOutput(); err != nil {
			return "", fmt.Errorf("docker start: %v: %s", err, out)
		}
	default:
		out, err := exec.Command("docker", "run", "-d", "--name", containerName,
			"-e", "POSTGRES_PASSWORD="+password, "-p", "127.0.0.1:"+hostPort+":5432", image).CombinedOutput()
		// Another test package may have created it at the same moment.
		if err != nil && !strings.Contains(string(out), "already in use") {
			return "", fmt.Errorf("docker run: %v: %s", err, out)
		}
	}

	url := "postgres://postgres:" + password + "@127.0.0.1:" + hostPort + "/postgres?sslmode=disable"
	deadline := time.Now().Add(30 * time.Second)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		c, err := pgx.Connect(ctx, url)
		cancel()
		if err == nil {
			c.Close(context.Background())
			return url, nil
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("test Postgres not ready: %v", err)
		}
		time.Sleep(300 * time.Millisecond)
	}
}

func withDatabase(url, name string) string {
	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		return url
	}
	// Rebuild the URL with the new database name, keeping everything else.
	q := ""
	if i := strings.Index(url, "?"); i >= 0 {
		q = url[i:]
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, name, q)
}

func randomSuffix() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
