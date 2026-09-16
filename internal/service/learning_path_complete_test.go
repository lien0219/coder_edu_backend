package service

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/util"

	"github.com/go-sql-driver/mysql"
)

func TestLearningPathCompletionModelDoesNotDeclareCompositeUniqueIndex(t *testing.T) {
	typ := reflect.TypeOf(model.LearningPathCompletion{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("gorm")
		if strings.Contains(tag, "uniqueIndex") {
			t.Fatalf("%s gorm tag must not auto-create unique index, got %q", field.Name, tag)
		}
	}
	user, ok := typ.FieldByName("UserID")
	if !ok || !strings.Contains(user.Tag.Get("gorm"), "index") {
		t.Fatal("UserID must keep ordinary index")
	}
	material, ok := typ.FieldByName("MaterialID")
	if !ok || !strings.Contains(material.Tag.Get("gorm"), "index") {
		t.Fatal("MaterialID must keep ordinary index")
	}
}

func TestCompleteMaterialDuplicateRequestAwardsXPOnce(t *testing.T) {
	svc, path, db := newPathBundle(t)
	paper, ver, qs := publishLinkedPaper(t, svc, db)
	mat := insertMaterial(t, path, "xp-mat", 2, 1, []string{"kp-pointer"})
	if err := db.Exec(`UPDATE learning_path_materials SET points = ? WHERE id = ?`, 15, mat.ID).Error; err != nil {
		t.Fatal(err)
	}
	userID := createTestUser(t, db, true)
	confirmWrongOn(t, svc, userID, paper, ver, qs, 0, "dup-xp")

	if err := path.CompleteMaterial(userID, mat.ID); err != nil {
		t.Fatal(err)
	}
	if err := path.CompleteMaterial(userID, mat.ID); err != nil {
		t.Fatal(err)
	}

	var xp int
	if err := db.Raw(`SELECT xp FROM users WHERE id = ?`, userID).Scan(&xp).Error; err != nil {
		t.Fatal(err)
	}
	if xp != 15 {
		t.Fatalf("xp=%d want 15", xp)
	}
	var completions int
	if err := db.Raw(`SELECT COUNT(*) FROM learning_path_completions WHERE user_id = ? AND material_id = ?`, userID, mat.ID).Scan(&completions).Error; err != nil {
		t.Fatal(err)
	}
	if completions != 1 {
		t.Fatalf("completions=%d", completions)
	}
	var logs int
	if err := db.Raw(`SELECT COUNT(*) FROM learning_logs WHERE user_id = ? AND activity = ?`, userID, "learning_path_complete").Scan(&logs).Error; err != nil {
		t.Fatal(err)
	}
	if logs != 1 {
		t.Fatalf("logs=%d", logs)
	}
}

func TestCompleteMaterialUniqueIndexRejectsSecondRow(t *testing.T) {
	svc, path, db := newPathBundle(t)
	paper, ver, qs := publishLinkedPaper(t, svc, db)
	mat := insertMaterial(t, path, "xp-mat", 2, 1, []string{"kp-pointer"})
	userID := createTestUser(t, db, true)
	confirmWrongOn(t, svc, userID, paper, ver, qs, 0, "uniq-row")
	if err := path.CompleteMaterial(userID, mat.ID); err != nil {
		t.Fatal(err)
	}
	dup := &model.LearningPathCompletion{UserID: userID, MaterialID: mat.ID}
	if err := path.Repo.CreateCompletion(dup); err == nil {
		t.Fatal("second completion row must hit unique index")
	} else if !isLearningPathCompletionConflict(err) {
		t.Fatalf("want completion unique conflict, got %v", err)
	}
}

func TestIsLearningPathCompletionConflictOnlyTargetIndex(t *testing.T) {
	target := errors.New("Error 1062 (23000): Duplicate entry '1-abc' for key 'learning_path_completions.idx_learning_path_completion_user_material'")
	if !isLearningPathCompletionConflict(target) {
		t.Fatal("target completion unique conflict must count as already completed")
	}
	sqlite := errors.New("UNIQUE constraint failed: learning_path_completions.user_id, learning_path_completions.material_id")
	if !isLearningPathCompletionConflict(sqlite) {
		t.Fatal("sqlite completion unique conflict must count as already completed")
	}
	otherUnique := errors.New("Error 1062 (23000): Duplicate entry 'a@b' for key 'users.uni_users_email'")
	if isLearningPathCompletionConflict(otherUnique) {
		t.Fatal("other unique conflicts must not be treated as already completed")
	}
	deadlock := errors.New("Error 1213 (40001): Deadlock found when trying to get lock; try restarting transaction")
	if isLearningPathCompletionConflict(deadlock) {
		t.Fatal("deadlock must not be swallowed")
	}
	lockWait := errors.New("Error 1205 (HY000): Lock wait timeout exceeded; try restarting transaction")
	if isLearningPathCompletionConflict(lockWait) {
		t.Fatal("lock wait timeout must not be swallowed")
	}
	conn := errors.New("invalid connection")
	if isLearningPathCompletionConflict(conn) {
		t.Fatal("connection errors must not be swallowed")
	}
	xp := util.ErrXPUpdateFailed
	if isLearningPathCompletionConflict(xp) {
		t.Fatal("xp failure must not be swallowed")
	}
	driverTarget := &mysql.MySQLError{
		Number:  1062,
		Message: "Duplicate entry '1-abc' for key 'learning_path_completions.idx_learning_path_completion_user_material'",
	}
	if !isLearningPathCompletionConflict(driverTarget) {
		t.Fatal("mysql 1062 on completion unique index must count as already completed")
	}
	driverOther := &mysql.MySQLError{
		Number:  1062,
		Message: "Duplicate entry 'a@b' for key 'users.uni_users_email'",
	}
	if isLearningPathCompletionConflict(driverOther) {
		t.Fatal("mysql 1062 on another index must not be swallowed")
	}
	if isLearningPathCompletionConflict(&mysql.MySQLError{Number: 1213, Message: "Deadlock found when trying to get lock"}) {
		t.Fatal("mysql deadlock must not be swallowed")
	}
}

func TestCompleteMaterialXPFailureRollsBack(t *testing.T) {
	svc, path, db := newPathBundle(t)
	paper, ver, qs := publishLinkedPaper(t, svc, db)
	mat := insertMaterial(t, path, "xp-mat", 2, 1, []string{"kp-pointer"})
	if err := db.Exec(`UPDATE learning_path_materials SET points = ? WHERE id = ?`, 20, mat.ID).Error; err != nil {
		t.Fatal(err)
	}
	userID := createTestUser(t, db, true)
	confirmWrongOn(t, svc, userID, paper, ver, qs, 0, "rollback-xp")

	if err := db.Exec(`
		CREATE TRIGGER deny_xp_update BEFORE UPDATE OF xp ON users
		BEGIN
			SELECT RAISE(ABORT, 'xp write failed');
		END
	`).Error; err != nil {
		t.Fatal(err)
	}

	err := path.CompleteMaterial(userID, mat.ID)
	if err == nil {
		t.Fatal("expected xp failure")
	}
	if errors.Is(err, util.ErrMaterialNotAccessible) || errors.Is(err, util.ErrResourceNotFound) {
		t.Fatalf("unexpected access error %v", err)
	}

	var completions int
	if err := db.Raw(`SELECT COUNT(*) FROM learning_path_completions WHERE user_id = ? AND material_id = ?`, userID, mat.ID).Scan(&completions).Error; err != nil {
		t.Fatal(err)
	}
	if completions != 0 {
		t.Fatalf("completion must roll back, got %d", completions)
	}
	var logs int
	if err := db.Raw(`SELECT COUNT(*) FROM learning_logs WHERE user_id = ? AND activity = ?`, userID, "learning_path_complete").Scan(&logs).Error; err != nil {
		t.Fatal(err)
	}
	if logs != 0 {
		t.Fatalf("learning log must roll back, got %d", logs)
	}
	var xp int
	if err := db.Raw(`SELECT xp FROM users WHERE id = ?`, userID).Scan(&xp).Error; err != nil {
		t.Fatal(err)
	}
	if xp != 0 {
		t.Fatalf("xp must stay 0, got %d", xp)
	}
}

func TestCompleteMaterialLogFailureRollsBack(t *testing.T) {
	svc, path, db := newPathBundle(t)
	paper, ver, qs := publishLinkedPaper(t, svc, db)
	mat := insertMaterial(t, path, "xp-mat", 2, 1, []string{"kp-pointer"})
	if err := db.Exec(`UPDATE learning_path_materials SET points = ? WHERE id = ?`, 8, mat.ID).Error; err != nil {
		t.Fatal(err)
	}
	userID := createTestUser(t, db, true)
	confirmWrongOn(t, svc, userID, paper, ver, qs, 0, "rollback-log")

	if err := db.Exec(`DROP TABLE learning_logs`).Error; err != nil {
		t.Fatal(err)
	}

	if err := path.CompleteMaterial(userID, mat.ID); err == nil {
		t.Fatal("expected log write failure")
	}
	var completions int
	if err := db.Raw(`SELECT COUNT(*) FROM learning_path_completions WHERE user_id = ? AND material_id = ?`, userID, mat.ID).Scan(&completions).Error; err != nil {
		t.Fatal(err)
	}
	if completions != 0 {
		t.Fatalf("completion must roll back when log fails, got %d", completions)
	}
	var xp int
	if err := db.Raw(`SELECT xp FROM users WHERE id = ?`, userID).Scan(&xp).Error; err != nil {
		t.Fatal(err)
	}
	if xp != 0 {
		t.Fatalf("xp must stay 0, got %d", xp)
	}
}
