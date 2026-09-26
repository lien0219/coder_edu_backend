package service

import (
	"errors"
	"testing"

	"github.com/go-sql-driver/mysql"
)

func TestLearningProfileAdminConflictDetectsUniqueKey(t *testing.T) {
	target := errors.New("Error 1062 (23000): Duplicate entry '1-2-pretest-platform_self_assessment' for key 'learning_profile_administrations.uk_lp_admin_user_inst_wave_program'")
	if !isLearningProfileAdminConflict(target) {
		t.Fatal("string 1062 on admin unique key must count as conflict")
	}
	driver := &mysql.MySQLError{
		Number:  1062,
		Message: "Duplicate entry '1-2-pretest-x' for key 'uk_lp_admin_user_inst_wave_program'",
	}
	if !isLearningProfileAdminConflict(driver) {
		t.Fatal("mysql 1062 on admin unique key must count as conflict")
	}
	if isLearningProfileAdminConflict(&mysql.MySQLError{Number: 1062, Message: "Duplicate entry 'a@b' for key 'users.uni_users_email'"}) {
		t.Fatal("mysql 1062 on another index must not be swallowed")
	}
	if isLearningProfileAnswerConflict(driver) {
		t.Fatal("admin unique must not match answer unique")
	}
}

func TestLearningProfileAnswerConflictDetectsUniqueKey(t *testing.T) {
	driver := &mysql.MySQLError{
		Number:  1062,
		Message: "Duplicate entry '9-12' for key 'uk_lp_answers_admin_item'",
	}
	if !isLearningProfileAnswerConflict(driver) {
		t.Fatal("answer unique 1062 must count as conflict")
	}
	if isLearningProfileAdminConflict(driver) {
		t.Fatal("answer unique must not match admin unique")
	}
}

func TestLearningProfileLockNameIsPerUserInstrumentWave(t *testing.T) {
	a := learningProfileLockName(7, "DL-C56-v1", "pretest")
	b := learningProfileLockName(8, "DL-C56-v1", "pretest")
	c := learningProfileLockName(7, "SDL-C20-v1", "pretest")
	if a == b || a == c {
		t.Fatalf("lock names collided %s %s %s", a, b, c)
	}
	if a != "lp-7-DL-C56-v1-pretest" {
		t.Fatalf("got %s", a)
	}
}
