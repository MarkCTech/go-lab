// schemagolden compares normalized mysqldump output to migrations/schema_golden.sql (same check as legacy bash script).
// Requires: Docker Compose stack running with healthy mysql; .env with DB_PASS and DB_NAME (run from repository tree).
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var autoIncRE = regexp.MustCompile(` AUTO_INCREMENT=[0-9]+`)

func main() {
	repo := findRepoRoot()
	_ = os.Chdir(repo)

	loadDotEnv(filepath.Join(repo, ".env"))

	dbPass := strings.TrimSpace(os.Getenv("DB_PASS"))
	dbName := strings.TrimSpace(os.Getenv("DB_NAME"))
	if dbPass == "" {
		fmt.Fprintln(os.Stderr, "schemagolden: DB_PASS not set (copy .env.example to .env)")
		os.Exit(1)
	}
	if dbName == "" {
		fmt.Fprintln(os.Stderr, "schemagolden: DB_NAME not set")
		os.Exit(1)
	}

	ignoreTable := fmt.Sprintf("%s.schema_migrations", dbName)
	args := []string{
		"compose", "exec", "-T",
		"-e", "MYSQL_PWD=" + dbPass,
		"mysql", "mysqldump", "-uroot",
		"--no-data", "--skip-comments", "--single-transaction",
		"--ignore-table=" + ignoreTable,
		dbName,
	}
	cmd := exec.Command("docker", args...)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "schemagolden: docker compose exec mysqldump: %v\n%s", err, stderr.String())
		os.Exit(1)
	}

	got := normalizeDump(out.Bytes())
	goldenPath := filepath.Join(repo, "migrations", "schema_golden.sql")
	wantRaw, err := os.ReadFile(goldenPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "schemagolden: read golden: %v\n", err)
		os.Exit(1)
	}
	want := normalizeDump(wantRaw)

	if !bytes.Equal(got, want) {
		fmt.Fprintln(os.Stderr, "Schema drift: normalized mysqldump differs from migrations/schema_golden.sql")
		fmt.Fprintln(os.Stderr, "If migrations changed intentionally, regenerate the golden file (see docs/migrations.md).")
		os.Exit(1)
	}
	fmt.Println("OK: schema matches migrations/schema_golden.sql")
}

func findRepoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		p := filepath.Join(dir, "migrations", "schema_golden.sql")
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	wd, _ := os.Getwd()
	return wd
}

func normalizeDump(b []byte) []byte {
	b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
	return autoIncRE.ReplaceAll(b, nil)
}

// loadDotEnv sets KEY=value from the first path that exists (minimal parser; no export syntax).
func loadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.IndexByte(line, '=')
		if i <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		val := strings.TrimSpace(line[i+1:])
		if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
			val = val[1 : len(val)-1]
		}
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}
