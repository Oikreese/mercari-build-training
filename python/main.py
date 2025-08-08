import os
import logging
import pathlib
from fastapi import FastAPI, Form, HTTPException, Depends, File, UploadFile, Query
from fastapi.responses import FileResponse
from fastapi.middleware.cors import CORSMiddleware
import sqlite3
from pydantic import BaseModel, Field
from typing import Optional
from contextlib import asynccontextmanager
import hashlib


# Define the path to the images & sqlite3 database
images = pathlib.Path(__file__).parent.resolve() / "images"
db = pathlib.Path(__file__).parent.resolve() / "db" / "mercari.sqlite3"
SQL_DB = pathlib.Path(__file__).parent.resolve() / "db" / "items.sql"

# Maximum file size (in bytes)
MAX_FILE_SIZE = 1024 * 1024  # 1MB (easier to find a picture to test)

def get_db():
    if not db.exists():
        setup_database()

    conn = sqlite3.connect(db, check_same_thread=False)
    conn.row_factory = sqlite3.Row  # Return rows as dictionaries
    try:
        yield conn
    finally:
        conn.close()


# STEP 5-1: set up the database connection
def setup_database():
    conn = sqlite3.connect(db)
    with open(SQL_DB, "r", encoding="utf-8") as f:
        conn.executescript(f.read())
    conn.commit()
    conn.close()
    logger.info("Database initialized successfully.")


@asynccontextmanager
async def lifespan(app: FastAPI):
    setup_database()
    yield


app = FastAPI(lifespan=lifespan)

logger = logging.getLogger("uvicorn")
logger.level = logging.DEBUG
images = pathlib.Path(__file__).parent.resolve() / "images"
origins = [os.environ.get("FRONT_URL", "http://localhost:3000")]
app.add_middleware(
    CORSMiddleware,
    allow_origins=origins,
    allow_credentials=False,
    allow_methods=["GET", "POST", "PUT", "DELETE"],
    allow_headers=["*"],
)


class HelloResponse(BaseModel):
    message: str


@app.get("/", response_model=HelloResponse)
def hello():
    return HelloResponse(**{"message": "Hello, world!"})


class AddItemResponse(BaseModel):
    message: str


# add_item is a handler to add a new item for POST /items .
@app.post("/items", response_model=AddItemResponse)
async def add_item(
    name: str = Form(...),
    category: str = Form(...),
    image: UploadFile = File(...),
    db: sqlite3.Connection = Depends(get_db)
):
    if not name or not category or not image:
        raise HTTPException(status_code=400, detail="name, category, and image are required")

    # Check file extension (when POST)
    if not image.filename.lower().endswith((".jpg", ".jpeg")):
        raise HTTPException(status_code=400, detail="Uploaded file must have a .jpg or .jpeg extension")

    # Check file size
    image_bytes = await image.read()
    if len(image_bytes) > MAX_FILE_SIZE:
        raise HTTPException(status_code=400, detail="File size exceeds the limit")

    # Validate MIME type
    image_type = image.content_type
    if image_type != "image/jpeg":
        raise HTTPException(status_code=400, detail="Uploaded file is not a valid JPG image")
    
    # Generate unique filename
    image_hash = hashlib.sha256(image_bytes).hexdigest()
    image_filename = f"{image_hash}.jpg"
    image_path = images / image_filename
    
    # Save file
    with open(image_path, "wb") as f:
        f.write(image_bytes)
    
    # Store item in database
    item = Item(name=name, category=category, image_name=image_filename)
    insert_item(item, db)
    
    return {"message": f"item received: {name}"}


# get_image is a handler to return an image for GET /images/{filename} .
@app.get("/images/{image_name}")
async def get_image(image_name):
    # Create image path
    image = images / image_name

    # Check file extension (when GET)
    if not image_name.endswith(".jpg"):
        raise HTTPException(status_code=400, detail="Image path does not end with .jpg")

    if not image.exists():
        logger.debug(f"Image not found: {image}")
        image = images / "default.jpg"

    return FileResponse(image)


class Item(BaseModel):
    id: Optional[int] = Field(None, description="Auto-generated ID")
    name: str
    category: str
    image_name: str
    category_name: Optional[str] = Field(None, description="Category name from categories table")


def insert_item(item: Item, db: sqlite3.Connection):
    # STEP 4-1: add an implementation to store an item
    try:
        cursor = db.cursor()
        cursor.execute("SELECT id FROM categories WHERE name = ?", (item.category,))
        category_row = cursor.fetchone()
        
        if category_row is None:
            cursor.execute("INSERT INTO categories (name) VALUES (?)", (item.category,))
            db.commit()  
            category_id = cursor.lastrowid
        else:
            category_id = category_row["id"]

        cursor.execute("SELECT id FROM items WHERE name = ?", (item.name,))
        existing_item = cursor.fetchone()

        if existing_item:
            logger.info(f"Item already exists: {existing_item}")
            raise HTTPException(status_code=400, detail="Item already exists")

        cursor.execute(
            "INSERT INTO items (name, category_id, image_name) VALUES (?, ?, ?)",
            (item.name, category_id, item.image_name),
        )
        db.commit()
        logger.info(f"New item inserted: {item.model_dump()}")

    except Exception as e:
        db.rollback()
        logger.error(f"Failed to save item: {e}")
        raise HTTPException(status_code=500, detail="Failed to save item")



@app.get("/items")
def get_items(db: sqlite3.Connection = Depends(get_db)):
    try:
        cursor = db.cursor()
        cursor.execute("""
            SELECT items.id, items.name, categories.name AS category_name, items.image_name
            FROM items
            JOIN categories ON items.category_id = categories.id
        """)
        items = cursor.fetchall()

        return {"items": [dict(row) for row in items]}
    except Exception as e:
        logger.error(f"Failed to get items: {e}")
        raise HTTPException(status_code=500, detail="Failed to get items")


@app.get("/items/{item_id}")
def get_item(item_id: int, db: sqlite3.Connection = Depends(get_db)):
    try:
        cursor = db.cursor()
        cursor.execute("""
            SELECT items.id, items.name, categories.name AS category_name, items.image_name
            FROM items
            JOIN categories ON items.category_id = categories.id
            WHERE items.id = ?
        """, (item_id,))
        item = cursor.fetchone()

        if item is None:
            raise HTTPException(status_code=404, detail="Item not found")

        return dict(item)
    except Exception as e:
        logger.error(f"Failed to get item {item_id}: {e}")
        raise HTTPException(status_code=500, detail="Failed to get item")


@app.get("/search")
def search_items(keyword: str = Query(..., min_length=1), db: sqlite3.Connection = Depends(get_db)):
    try:
        cursor = db.cursor()
        cursor.execute("""
            SELECT items.name, categories.name AS category_name, items.image_name
            FROM items
            JOIN categories ON items.category_id = categories.id
            WHERE items.name LIKE ? OR categories.name LIKE ?
        """, (f"%{keyword}%", f"%{keyword}%"))
        items = cursor.fetchall()

        return {
            "items": [
                {"name": row["name"], "category": row["category_name"], "image_name": row["image_name"]}
                for row in items
            ]
        }
    except Exception as e:
        logger.error(f"Failed to search items: {e}")
        raise HTTPException(status_code=500, detail="Failed to search items")


@app.delete("/items/{item_id}")
def delete_item(item_id: int, db: sqlite3.Connection = Depends(get_db)):
    try:
        cursor = db.cursor()
        cursor.execute("SELECT image_name FROM items WHERE id = ?", (item_id,))
        item = cursor.fetchone()

        if item is None:
            raise HTTPException(status_code=404, detail="Item not found")

        image_name = item["image_name"]
        image_path = images / image_name

        cursor.execute("DELETE FROM items WHERE id = ?", (item_id,))
        db.commit()

        cursor.execute("SELECT COUNT(*) FROM items WHERE image_name = ?", (image_name,))
        count = cursor.fetchone()[0]
        
        if count == 0 and image_path.exists():
            os.remove(image_path)

        return {"message": f"Item {item_id} deleted successfully"}

    except Exception as e:
        db.rollback()
        logger.error(f"Failed to delete item {item_id}: {e}")
        raise HTTPException(status_code=500, detail="Failed to delete item")


@app.patch("/items/{item_id}")
async def update_item(
    item_id: int,
    name: Optional[str] = Form(None),
    category: Optional[str] = Form(None),
    image: Optional[UploadFile] = File(None),
    db: sqlite3.Connection = Depends(get_db)
):
    try:
        cursor = db.cursor()

        cursor.execute("""
            SELECT items.id, items.name, categories.name AS category_name, items.image_name
            FROM items
            JOIN categories ON items.category_id = categories.id
            WHERE items.id = ?
        """, (item_id,))
        existing_item = cursor.fetchone()

        if existing_item is None:
            raise HTTPException(status_code=404, detail="Item not found")

        update_fields = []
        update_values = []

        if name and name != existing_item["name"]:
            update_fields.append("name = ?")
            update_values.append(name)

        if category and category != existing_item["category_name"]:
            cursor.execute("SELECT id FROM categories WHERE name = ?", (category,))
            category_row = cursor.fetchone()

            if category_row is None:
                cursor.execute("INSERT INTO categories (name) VALUES (?)", (category,))
                db.commit()
                category_id = cursor.lastrowid
            else:
                category_id = category_row["id"]

            update_fields.append("category_id = ?")
            update_values.append(category_id)

        if image:
            if not image.filename.lower().endswith((".jpg", ".jpeg")):
                raise HTTPException(status_code=400, detail="Uploaded file must have a .jpg or .jpeg extension")

            image_bytes = await image.read()
            if len(image_bytes) > MAX_FILE_SIZE:
                raise HTTPException(status_code=400, detail="File size exceeds the limit")

            image_hash = hashlib.sha256(image_bytes).hexdigest()
            image_filename = f"{image_hash}.jpg"
            image_path = images / image_filename

            if image_filename != existing_item["image_name"]:
                with open(image_path, "wb") as f:
                    f.write(image_bytes)

                if not image_path.exists():
                    logger.error(f"Failed to save image: {image_path}")
                    raise HTTPException(status_code=500, detail="Failed to save image")
                else:
                    logger.info(f"New image saved: {image_path}")

                    old_image_path = images / existing_item["image_name"]
                    if old_image_path.exists():
                        old_image_path.unlink()

                update_fields.append("image_name = ?")
                update_values.append(image_filename)
                
            elif image_filename == existing_item["image_name"] and not image_path.exists():
                with open(image_path, "wb") as f:
                    f.write(image_bytes)

                if not image_path.exists():
                    logger.error(f"Failed to save image: {image_path}")
                    raise HTTPException(status_code=500, detail="Failed to save image")
                else:
                    logger.info(f"New image saved: {image_path}")
                    update_fields.append("image_name = ?")
                    update_values.append(image_filename)
        
            else:
                logger.info(f"Image already exists: {image_path}")

        if not update_fields:
            return {"message": "No changes detected, item not updated"}

        update_values.append(item_id)
        update_query = f"UPDATE items SET {', '.join(update_fields)} WHERE id = ?"
        cursor.execute(update_query, update_values)
        db.commit()

        return {"message": "Item updated successfully"}
    
    except Exception as e:
        db.rollback()
        logger.error(f"Failed to update item {item_id}: {e}")
        raise HTTPException(status_code=500, detail="Failed to update item")
