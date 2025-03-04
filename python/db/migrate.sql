BEGIN TRANSACTION;

CREATE TEMP TABLE items_old AS
SELECT id, name, category, image_name FROM items;

INSERT INTO categories (name)
SELECT DISTINCT category FROM items_old;

CREATE TABLE items_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    category_id INTEGER NOT NULL,
    image_name TEXT NOT NULL,
    FOREIGN KEY (category_id) REFERENCES categories(id)
);

INSERT INTO items_new (id, name, category_id, image_name)
SELECT i.id, i.name, c.id, i.image_name
FROM items_old i
JOIN categories c ON i.category = c.name;

DROP TABLE items;

ALTER TABLE items_new RENAME TO items;

DROP TABLE items_old;

COMMIT;
