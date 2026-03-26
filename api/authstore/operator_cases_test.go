package authstore

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCreateOperatorCaseRejectsInvalidPriority(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	s := New(db, 0, 0)
	_, err = s.CreateOperatorCase(context.Background(), 10, nil, "Title", "desc", "invalid", 1)
	if !errors.Is(err, ErrInvalidCasePriority) {
		t.Fatalf("expected ErrInvalidCasePriority, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected SQL calls: %v", err)
	}
}

func TestUpdateOperatorCaseRejectsInvalidStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	s := New(db, 0, 0)
	status := "bad"
	err = s.UpdateOperatorCase(context.Background(), 1, &status, nil, nil)
	if !errors.Is(err, ErrInvalidCaseStatus) {
		t.Fatalf("expected ErrInvalidCaseStatus, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected SQL calls: %v", err)
	}
}

func TestUpdateOperatorCaseRejectsInvalidPriority(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	s := New(db, 0, 0)
	priority := "bad"
	err = s.UpdateOperatorCase(context.Background(), 1, nil, &priority, nil)
	if !errors.Is(err, ErrInvalidCasePriority) {
		t.Fatalf("expected ErrInvalidCasePriority, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected SQL calls: %v", err)
	}
}
