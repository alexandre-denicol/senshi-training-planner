package blocks

import (
	"context"
	"errors"
	"time"
)

const MaxNameRunes = 120

var (
	ErrInvalidRequest  = errors.New("invalid request")
	ErrInvalidCategory = errors.New("invalid category")
	ErrNameExists      = errors.New("block name already exists in category")
	ErrNotFound        = errors.New("block not found")
	ErrInUse           = errors.New("block is in use")
)

type CategoryRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Block struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Active    bool        `json:"active"`
	Category  CategoryRef `json:"category"`
	CreatedAt time.Time   `json:"createdAt"`
	UpdatedAt time.Time   `json:"updatedAt"`
}

type CreateInput struct {
	Name       string `json:"name"`
	CategoryID string `json:"categoryId"`
}

type UpdateInput struct {
	Name       string `json:"name"`
	CategoryID string `json:"categoryId"`
}

type StatusInput struct {
	Active *bool `json:"active"`
}

type NewBlock struct {
	ID         string
	Name       string
	CategoryID string
}

type Store interface {
	ListBlocks(ctx context.Context) ([]Block, error)
	CreateBlock(ctx context.Context, block NewBlock) (Block, error)
	UpdateBlock(ctx context.Context, id string, name string, categoryID string) (Block, error)
	SetBlockStatus(ctx context.Context, id string, active bool) (Block, error)
	DeleteBlock(ctx context.Context, id string) error
}

type ServiceAPI interface {
	List(ctx context.Context) ([]Block, error)
	Create(ctx context.Context, input CreateInput) (Block, error)
	Update(ctx context.Context, id string, input UpdateInput) (Block, error)
	SetStatus(ctx context.Context, id string, input StatusInput) (Block, error)
	Delete(ctx context.Context, id string) error
}
