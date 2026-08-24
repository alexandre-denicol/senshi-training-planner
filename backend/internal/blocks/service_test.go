package blocks

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

const (
	blockID            = "11111111-1111-1111-1111-111111111111"
	missingBlockID     = "22222222-2222-2222-2222-222222222222"
	activeCategoryID   = "33333333-3333-3333-3333-333333333333"
	otherCategoryID    = "44444444-4444-4444-4444-444444444444"
	inactiveCategoryID = "55555555-5555-5555-5555-555555555555"
)

func TestServiceCreateBlock(t *testing.T) {
	store := &fakeBlockStore{}
	service := NewService(store)
	service.newUUID = func() (string, error) { return blockID, nil }

	block, err := service.Create(context.Background(), CreateInput{Name: "  Base  ", CategoryID: activeCategoryID})
	if err != nil {
		t.Fatalf("expected create success, got %v", err)
	}
	if block.ID != blockID {
		t.Fatalf("expected generated id, got %s", block.ID)
	}
	if store.created[0].Name != "Base" {
		t.Fatalf("expected trimmed name, got %q", store.created[0].Name)
	}
	if store.created[0].CategoryID != activeCategoryID {
		t.Fatalf("expected category id %s, got %s", activeCategoryID, store.created[0].CategoryID)
	}
	if !block.Active {
		t.Fatal("expected new block to be active")
	}
}

func TestServiceCreateBlockWithDescriptionAndSequence(t *testing.T) {
	store := &fakeBlockStore{}
	service := NewService(store)
	service.newUUID = func() (string, error) { return blockID, nil }

	description := "  Executar em dupla.\nPreservar distância.  "
	block, err := service.Create(context.Background(), CreateInput{
		Name:        "Combinação 01",
		CategoryID:  activeCategoryID,
		Description: &description,
		Sequence: []SequenceItemInput{
			{Text: "  Jab  "},
			{Text: "Direto"},
			{Text: "Jab"},
			{Text: "30 segundos de manopla"},
		},
	})
	if err != nil {
		t.Fatalf("expected create success, got %v", err)
	}
	if store.created[0].Description == nil || *store.created[0].Description != "Executar em dupla.\nPreservar distância." {
		t.Fatalf("expected trimmed description, got %#v", store.created[0].Description)
	}
	if len(store.created[0].Sequence) != 4 {
		t.Fatalf("expected sequence items, got %#v", store.created[0].Sequence)
	}
	if store.created[0].Sequence[0].Text != "Jab" || store.created[0].Sequence[0].Position != 1 {
		t.Fatalf("expected ordered trimmed sequence, got %#v", store.created[0].Sequence)
	}
	if block.Sequence[2].Text != "Jab" {
		t.Fatalf("expected duplicate text to be allowed, got %#v", block.Sequence)
	}
}

func TestServiceNormalizesWhitespaceOnlyDescriptionToNil(t *testing.T) {
	store := &fakeBlockStore{}
	service := NewService(store)
	service.newUUID = func() (string, error) { return blockID, nil }
	description := " \n\t "

	if _, err := service.Create(context.Background(), CreateInput{Name: "Base", CategoryID: activeCategoryID, Description: &description}); err != nil {
		t.Fatalf("expected create success, got %v", err)
	}
	if store.created[0].Description != nil {
		t.Fatalf("expected whitespace-only description to become nil, got %#v", store.created[0].Description)
	}
}

func TestServiceRejectsInvalidBlockInput(t *testing.T) {
	service := NewService(&fakeBlockStore{})

	tests := []struct {
		name  string
		input CreateInput
		err   error
	}{
		{name: "empty name", input: CreateInput{Name: "", CategoryID: activeCategoryID}, err: ErrInvalidRequest},
		{name: "whitespace name", input: CreateInput{Name: "   \t\n", CategoryID: activeCategoryID}, err: ErrInvalidRequest},
		{name: "too long name", input: CreateInput{Name: strings.Repeat("á", MaxNameRunes+1), CategoryID: activeCategoryID}, err: ErrInvalidRequest},
		{name: "invalid category uuid", input: CreateInput{Name: "Base", CategoryID: "not-a-uuid"}, err: ErrInvalidCategory},
		{name: "too long description", input: CreateInput{Name: "Base", CategoryID: activeCategoryID, Description: stringPtr(strings.Repeat("á", MaxDescriptionRunes+1))}, err: ErrInvalidRequest},
		{name: "blank sequence item", input: CreateInput{Name: "Base", CategoryID: activeCategoryID, Sequence: []SequenceItemInput{{Text: "   "}}}, err: ErrInvalidRequest},
		{name: "too long sequence item", input: CreateInput{Name: "Base", CategoryID: activeCategoryID, Sequence: []SequenceItemInput{{Text: strings.Repeat("á", MaxSequenceTextRunes+1)}}}, err: ErrInvalidRequest},
		{name: "too many sequence items", input: CreateInput{Name: "Base", CategoryID: activeCategoryID, Sequence: repeatSequenceItems(MaxSequenceItems + 1)}, err: ErrInvalidRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Create(context.Background(), tt.input)
			if !errors.Is(err, tt.err) {
				t.Fatalf("expected %v, got %v", tt.err, err)
			}
		})
	}
}

func TestServiceRejectsInvalidCategoriesAndDuplicateNames(t *testing.T) {
	store := &fakeBlockStore{createErr: ErrInvalidCategory}
	service := NewService(store)
	service.newUUID = func() (string, error) { return blockID, nil }

	_, err := service.Create(context.Background(), CreateInput{Name: "Base", CategoryID: activeCategoryID})
	if !errors.Is(err, ErrInvalidCategory) {
		t.Fatalf("expected invalid category for nonexistent category, got %v", err)
	}

	store.createErr = ErrInvalidCategory
	_, err = service.Create(context.Background(), CreateInput{Name: "Base", CategoryID: inactiveCategoryID})
	if !errors.Is(err, ErrInvalidCategory) {
		t.Fatalf("expected invalid category for inactive category, got %v", err)
	}

	store.createErr = ErrNameExists
	_, err = service.Create(context.Background(), CreateInput{Name: "Base", CategoryID: activeCategoryID})
	if !errors.Is(err, ErrNameExists) {
		t.Fatalf("expected duplicate name, got %v", err)
	}
}

func TestServiceAllowsSameNameInDifferentCategories(t *testing.T) {
	store := &fakeBlockStore{}
	service := NewService(store)
	ids := []string{blockID, missingBlockID}
	service.newUUID = func() (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	}

	if _, err := service.Create(context.Background(), CreateInput{Name: "Base", CategoryID: activeCategoryID}); err != nil {
		t.Fatalf("expected first create success, got %v", err)
	}
	if _, err := service.Create(context.Background(), CreateInput{Name: "Base", CategoryID: otherCategoryID}); err != nil {
		t.Fatalf("expected same name in another category to be allowed, got %v", err)
	}
	if len(store.created) != 2 {
		t.Fatalf("expected two create attempts, got %d", len(store.created))
	}
}

func TestServiceUpdateStatusDeleteAndNotFound(t *testing.T) {
	store := &fakeBlockStore{}
	service := NewService(store)

	updated, err := service.Update(context.Background(), blockID, UpdateInput{Name: "  Avançado  ", CategoryID: otherCategoryID})
	if err != nil {
		t.Fatalf("expected update success, got %v", err)
	}
	if updated.Name != "Avançado" {
		t.Fatalf("expected trimmed name, got %q", updated.Name)
	}
	if store.updateCategoryID != otherCategoryID {
		t.Fatalf("expected move to category %s, got %s", otherCategoryID, store.updateCategoryID)
	}

	store.updateErr = ErrInvalidCategory
	_, err = service.Update(context.Background(), blockID, UpdateInput{Name: "Base", CategoryID: inactiveCategoryID})
	if !errors.Is(err, ErrInvalidCategory) {
		t.Fatalf("expected invalid category on inactive move, got %v", err)
	}
	store.updateErr = nil

	active := false
	inactive, err := service.SetStatus(context.Background(), blockID, StatusInput{Active: &active})
	if err != nil {
		t.Fatalf("expected deactivate success, got %v", err)
	}
	if inactive.Active {
		t.Fatal("expected inactive block")
	}

	active = true
	reactivated, err := service.SetStatus(context.Background(), blockID, StatusInput{Active: &active})
	if err != nil {
		t.Fatalf("expected reactivate success, got %v", err)
	}
	if !reactivated.Active {
		t.Fatal("expected active block")
	}

	if err := service.Delete(context.Background(), blockID); err != nil {
		t.Fatalf("expected delete success, got %v", err)
	}
	if store.deletedID != blockID {
		t.Fatalf("expected deleted id %s, got %s", blockID, store.deletedID)
	}

	_, err = service.Update(context.Background(), missingBlockID, UpdateInput{Name: "Nome", CategoryID: activeCategoryID})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected missing update not found, got %v", err)
	}
	if err := service.Delete(context.Background(), missingBlockID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected missing delete not found, got %v", err)
	}
	store.deleteErr = ErrInUse
	if err := service.Delete(context.Background(), blockID); !errors.Is(err, ErrInUse) {
		t.Fatalf("expected in-use delete error, got %v", err)
	}
	_, err = service.SetStatus(context.Background(), blockID, StatusInput{})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected missing active invalid request, got %v", err)
	}
}

type fakeBlockStore struct {
	created          []NewBlock
	createErr        error
	updateErr        error
	deleteErr        error
	updateCategoryID string
	deletedID        string
}

func (s *fakeBlockStore) ListBlocks(context.Context) ([]Block, error) {
	return nil, nil
}

func (s *fakeBlockStore) CreateBlock(_ context.Context, block NewBlock) (Block, error) {
	s.created = append(s.created, block)
	if s.createErr != nil {
		return Block{}, s.createErr
	}
	created := blockFixture(block.ID, block.Name, block.CategoryID, true)
	created.Description = block.Description
	created.Sequence = sequenceItemsFromNew(block.Sequence)
	return created, nil
}

func (s *fakeBlockStore) UpdateBlock(_ context.Context, id string, name string, categoryID string, description *string, sequence []NewSequenceItem) (Block, error) {
	if id == missingBlockID {
		return Block{}, ErrNotFound
	}
	if s.updateErr != nil {
		return Block{}, s.updateErr
	}
	s.updateCategoryID = categoryID
	block := blockFixture(id, name, categoryID, true)
	block.Description = description
	block.Sequence = sequenceItemsFromNew(sequence)
	return block, nil
}

func (s *fakeBlockStore) SetBlockStatus(_ context.Context, id string, active bool) (Block, error) {
	if id == missingBlockID {
		return Block{}, ErrNotFound
	}
	return blockFixture(id, "Base", activeCategoryID, active), nil
}

func stringPtr(value string) *string {
	return &value
}

func repeatSequenceItems(total int) []SequenceItemInput {
	items := make([]SequenceItemInput, total)
	for index := range items {
		items[index] = SequenceItemInput{Text: "Item"}
	}
	return items
}

func (s *fakeBlockStore) DeleteBlock(_ context.Context, id string) error {
	if id == missingBlockID {
		return ErrNotFound
	}
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deletedID = id
	return nil
}

func blockFixture(id string, name string, categoryID string, active bool) Block {
	now := time.Now().UTC()
	return Block{
		ID:     id,
		Name:   name,
		Active: active,
		Category: CategoryRef{
			ID:   categoryID,
			Name: "Categoria",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}
