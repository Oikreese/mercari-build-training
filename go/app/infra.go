package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

var errImageNotFound = errors.New("image not found")
var errItemNotFound = errors.New("item not found")

type Item struct {
	ID   int    `db:"id" json:"-"`
	Name string `db:"name" json:"name"`
	Category string `db:"-" json:"category"`
	ImageName string `db:"image_name" json:"image_name"`
}

// Please run `go generate ./...` to generate the mock implementation
// ItemRepository is an interface to manage items.
//
//go:generate go run go.uber.org/mock/mockgen -source=$GOFILE -package=${GOPACKAGE} -destination=./mock_$GOFILE
type ItemRepository interface {
	Insert(ctx context.Context, item *Item) error
	GetAll(ctx context.Context) ([]Item, error)
	GetByID(ctx context.Context, item_id string) (Item, error)
	Search(ctx context.Context, keyword string) ([]Item, error)
}

// itemRepository is an implementation of ItemRepository
type itemRepository struct {
	// fileName is the path to the JSON file storing items.
	//fileName string
	// filePath is the absolute path to the JSON file
	//filePath string
	DB *sql.DB
}

// NewItemRepository creates a new itemRepository.
func NewItemRepository(db *sql.DB) (ItemRepository, error) {
	sqlBytes, err := os.ReadFile("db/items.sql")
	if err != nil {
		slog.Error("failed to read schema file: ", "error", err)
		return nil, err
	}

	_, err = db.Exec(string(sqlBytes))
	if err != nil {
		slog.Error("failed to execute schema file: ", "error", err)
		return nil, err
	}
	return &itemRepository{DB: db}, nil
}

// Insert inserts an item into the repository.
func (i *itemRepository) Insert(ctx context.Context, item *Item) error {
	tx, err := i.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// Check if the item already exists
	var existingItem Item
	err = tx.QueryRowContext(ctx, "SELECT id FROM items WHERE name = ? AND category_id = (SELECT id FROM categories WHERE name = ?)", item.Name, item.Category).Scan(&existingItem.ID)
	if err == nil {
		slog.Info("item already exists", "name", item.Name, "category", item.Category)
		tx.Rollback()
		return nil
	} else if err != sql.ErrNoRows {
		tx.Rollback()
		return err
	}

	var categoryID int
	err = tx.QueryRowContext(ctx, "SELECT id FROM categories WHERE name = ?", item.Category).Scan(&categoryID)
	if err != nil {
		if err == sql.ErrNoRows {
			result, err := tx.ExecContext(ctx, "INSERT INTO categories (name) VALUES (?)", item.Category)
			if err != nil {
				tx.Rollback()
				return err
			}
			lastID, err := result.LastInsertId()
			if err != nil {
				tx.Rollback()
				return err
			}
			categoryID = int(lastID)
		} else {
			tx.Rollback()
			return err
		}
	}

	_, err = tx.ExecContext(ctx,
		"INSERT INTO items (name, category_id, image_name) VALUES (?, ?, ?)",
		item.Name, categoryID, item.ImageName)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// StoreImage stores an image and returns an error if any.
// This package doesn't have a related interface for simplicity.
func StoreImage(fileName string, image []byte) error {
	// STEP 4-4: add an implementation to store an image
	dir := "images"

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create image directory: %w", err)
	}

	filePath := filepath.Join(dir, fileName)

	if err := os.WriteFile(filePath, image, 0644); err != nil {
		return fmt.Errorf("failed to write image file: %w", err)
	}

	return nil
}


func (i *itemRepository) GetAll(ctx context.Context) ([]Item, error) {
	query := `
		SELECT items.id, items.name, categories.name AS category_name, items.image_name
		FROM items
		JOIN categories ON items.category_id = categories.id
		`
	rows, err := i.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.ImageName)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

func (i *itemRepository) GetByID(ctx context.Context, item_id string) (Item, error) {
	query := `
		SELECT items.id, items.name, categories.name AS category_name, items.image_name
		FROM items
		JOIN categories ON items.category_id = categories.id
		WHERE items.id = ?
		`
	row := i.DB.QueryRow(query, item_id)
	var item Item
	err := row.Scan(&item.ID, &item.Name, &item.Category, &item.ImageName)
	if err != nil {
		if err == sql.ErrNoRows {
			return Item{}, errItemNotFound
		} else {
			return Item{}, err
		}
	}
	return item, nil
}

func (i *itemRepository) Search(ctx context.Context, keyword string) ([]Item, error) {
	query := `
        SELECT items.name, categories.name AS category_name, items.image_name
		FROM items
		JOIN categories ON items.category_id = categories.id
		WHERE items.name LIKE ? OR categories.name LIKE ?
        `
	rows, err := i.DB.Query(query, "%"+keyword+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.ImageName)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}