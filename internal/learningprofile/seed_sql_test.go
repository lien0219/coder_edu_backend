package learningprofile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func seedSQL(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "migrations", "20260923_learning_profile_seed.sql"))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestSeedSQLSafetyStatic(t *testing.T) {
	text := seedSQL(t)
	if !strings.Contains(text, "START TRANSACTION") || !strings.Contains(text, "COMMIT") {
		t.Fatal("seed must declare a DML transaction")
	}
	if !strings.Contains(text, "CREATE TEMPORARY TABLE expected_items") {
		t.Fatal("seed must stage expected items for conflict checks")
	}
	if !strings.Contains(text, "lp_seed_guard") || !strings.Contains(text, "CHECK (allowed = 1)") {
		t.Fatal("seed must abort on disallowed recovery paths")
	}
	if !strings.Contains(text, "CREATE PROCEDURE seed_learning_profile_official_v1") {
		t.Fatal("seed DML must run inside a procedure so SQLEXCEPTION can roll back")
	}
	if !strings.Contains(text, "ROLLBACK") || !strings.Contains(text, "RESIGNAL") {
		t.Fatal("seed procedure must roll back and rethrow on error")
	}

	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		upper := strings.ToUpper(trimmed)
		if strings.Contains(upper, "INSERT IGNORE") || strings.Contains(upper, "REPLACE INTO") || strings.Contains(upper, "ON DUPLICATE KEY") {
			t.Fatalf("destructive upsert in executable SQL: %s", trimmed)
		}
		if strings.HasPrefix(upper, "DELETE ") || strings.HasPrefix(upper, "UPDATE ") || strings.HasPrefix(upper, "TRUNCATE") {
			t.Fatalf("seed executable SQL must not rewrite live rows: %s", trimmed)
		}
		if strings.HasPrefix(upper, "DROP TABLE") && !strings.Contains(upper, "TEMPORARY") {
			t.Fatalf("seed must not drop durable tables: %s", trimmed)
		}
	}
}

func TestSeedSQLContainsOfficialStemsAndOptions(t *testing.T) {
	text := seedSQL(t)
	if strings.Contains(text, "性别") || strings.Contains(text, "KMO") {
		t.Fatal("seed must not include demographics or appendix")
	}
	for _, item := range OfficialItems() {
		if !strings.Contains(text, item.ItemCode) {
			t.Fatalf("seed missing code %s", item.ItemCode)
		}
		if !strings.Contains(text, item.Stem) {
			t.Fatalf("seed missing stem for %s", item.ItemCode)
		}
	}
	if !strings.Contains(text, `"label":"非常不符合"`) || !strings.Contains(text, `"value":5`) {
		t.Fatal("seed missing DL option labels/values")
	}
	if !strings.Contains(text, `"label":"非常不同意"`) || !strings.Contains(text, `"label":"一般 / 不确定"`) || !strings.Contains(text, `"value":7`) {
		t.Fatal("seed missing SDL option labels/values")
	}
}
