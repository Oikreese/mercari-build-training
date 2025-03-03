import os
import logging
import pathlib
from fastapi import FastAPI, Form, HTTPException, File, UploadFile
from fastapi.responses import FileResponse
from fastapi.middleware.cors import CORSMiddleware
import sqlite3
from pydantic import BaseModel
from contextlib import asynccontextmanager
import json
import hashlib


# Define the path to the images & sqlite3 database
images = pathlib.Path(__file__).parent.resolve() / "images"
db = pathlib.Path(__file__).parent.resolve() / "db" / "mercari.sqlite3"
JSON_DB = pathlib.Path(__file__).parent.resolve() / "db" / "items.json"


def get_db():
    if not db.exists():
        yield

    conn = sqlite3.connect(db)
    conn.row_factory = sqlite3.Row  # Return rows as dictionaries
    try:
        yield conn
    finally:
        conn.close()


# STEP 5-1: set up the database connection
def setup_database():
    pass


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
    image: UploadFile = File(...)
):
    if not name or not category or not image:
        raise HTTPException(status_code=400, detail="name, category, and image are required")

    image_bytes = await image.read()
    image_hash = hashlib.sha256(image_bytes).hexdigest()
    image_filename = f"{image_hash}.jpg"
    image_path = pathlib.Path(__file__).parent.resolve() / "images" / image_filename

    with open(image_path, "wb") as f:
        f.write(image_bytes)

    item = Item(name=name, category=category, image_name=image_filename)
    insert_item(item, image_filename)

    return {"message": f"item received: {name}"}


# get_image is a handler to return an image for GET /images/{filename} .
@app.get("/image/{image_name}")
async def get_image(image_name):
    # Create image path
    image = images / image_name

    if not image_name.endswith(".jpg"):
        raise HTTPException(status_code=400, detail="Image path does not end with .jpg")

    if not image.exists():
        logger.debug(f"Image not found: {image}")
        image = images / "default.jpg"

    return FileResponse(image)


class Item(BaseModel):
    name: str
    category: str
    image_name: str


def insert_item(item: Item, image_name: str):
    # STEP 4-1: add an implementation to store an item
    try:
        if not JSON_DB.exists():
            with open(JSON_DB, "w", encoding="utf-8") as f:
                json.dump({"items": []}, f, indent=2)

        with open(JSON_DB, "r+", encoding="utf-8") as f:
            content = f.read().strip()
            data = json.loads(content) if content else {"items": []}
            logger.info("Succeeded to open json file")

            if "items" not in data:
                data["items"] = []

            existing_item = next((i for i in data["items"] if i["name"] == item.name), None)

            if existing_item:
                logger.info(f"Item already exists, updating image_name: {existing_item}")
                existing_item["image_name"] = image_name
            else:
                new_item = item.dict()
                new_item["image_name"] = image_name
                data["items"].append(new_item)
                logger.info(f"New item inserted: {new_item}")

            f.seek(0)
            json.dump(data, f, indent=2, ensure_ascii=False)
            f.truncate()

    except Exception as e:
        logger.error(f"Failed to save item: {e}")
        raise HTTPException(status_code=500, detail="Failed to save item")



@app.get("/items")
@app.get("/items/{item_id}")
def get_items(item_id: int = None):
    try:
        if not JSON_DB.exists():
            return {"items": []}

        with open(JSON_DB, "r", encoding="utf-8") as f:
            content = f.read().strip()
            data = json.loads(content) if content else {"items": []}
        
        items = data.get("items", [])

        if item_id is None:
            return {"items": items}

        if item_id < 1 or item_id > len(items):
            raise HTTPException(status_code=404, detail="Item not found")

        return items[item_id - 1]

    except Exception as e:
        logger.error(f"Failed to get items: {e}")
        raise HTTPException(status_code=500, detail="Failed to get items")