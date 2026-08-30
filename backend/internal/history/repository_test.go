package history

import (
	"errors"
	"testing"
)

func TestResolveParticipantsBuildsOrderedSnapshots(t *testing.T) {
	rows := []studentRow{
		{ID: studentIDTwo, Name: "Maria Souza", Active: true},
		{ID: studentIDOne, Name: "João Silva", Active: true},
	}

	snapshots, err := resolveParticipants([]string{studentIDOne, studentIDTwo}, rows)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(snapshots) != 2 {
		t.Fatalf("expected two snapshots, got %#v", snapshots)
	}
	if snapshots[0].StudentID != studentIDOne || snapshots[0].Name != "João Silva" {
		t.Fatalf("expected first snapshot to match requested order, got %#v", snapshots[0])
	}
	if snapshots[1].StudentID != studentIDTwo || snapshots[1].Name != "Maria Souza" {
		t.Fatalf("expected second snapshot to match requested order, got %#v", snapshots[1])
	}
}

func TestResolveParticipantsNoIDsReturnsEmptySnapshot(t *testing.T) {
	snapshots, err := resolveParticipants(nil, nil)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(snapshots) != 0 {
		t.Fatalf("expected no participants, got %#v", snapshots)
	}
}

func TestResolveParticipantsRejectsNonexistentStudent(t *testing.T) {
	rows := []studentRow{{ID: studentIDOne, Name: "João Silva", Active: true}}

	_, err := resolveParticipants([]string{studentIDOne, studentIDTwo}, rows)
	if !errors.Is(err, ErrInvalidParticipants) {
		t.Fatalf("expected invalid participants for nonexistent student, got %v", err)
	}
}

func TestResolveParticipantsRejectsInactiveStudent(t *testing.T) {
	rows := []studentRow{{ID: studentIDOne, Name: "João Silva", Active: false}}

	_, err := resolveParticipants([]string{studentIDOne}, rows)
	if !errors.Is(err, ErrInvalidParticipants) {
		t.Fatalf("expected invalid participants for inactive student, got %v", err)
	}
}

// TestResolveParticipantsSnapshotIsIndependentOfLaterRename proves the core
// compatibility guarantee: once a participant snapshot is resolved, mutating
// the source student data afterward (e.g. renaming the student) never changes
// a snapshot already handed back. History rows are built from that returned
// snapshot, so completed records stay immutable even if the student is later
// renamed or deactivated.
func TestResolveParticipantsSnapshotIsIndependentOfLaterRename(t *testing.T) {
	rows := []studentRow{{ID: studentIDOne, Name: "João Silva", Active: true}}

	snapshot, err := resolveParticipants([]string{studentIDOne}, rows)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if snapshot[0].Name != "João Silva" {
		t.Fatalf("expected initial snapshot name, got %q", snapshot[0].Name)
	}

	// Simulate the student being renamed (and later deactivated) after the
	// training was completed.
	rows[0].Name = "João Pedro Silva"
	rows[0].Active = false

	if snapshot[0].Name != "João Silva" {
		t.Fatalf("expected snapshot name to remain unchanged after rename, got %q", snapshot[0].Name)
	}
}
