package app

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type Server struct {
	// Port is the port number to listen on.
	Port string
	// ImageDirPath is the path to the directory storing images.
	ImageDirPath string
}

// Run is a method to start the server.
// This method returns 0 if the server started successfully, and 1 otherwise.
func (s Server) Run() int {
	// set up logger
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)
	// STEP 4-6: set the log level to DEBUG
	slog.SetLogLoggerLevel(slog.LevelDebug)

	// set up CORS settings
	frontURL, found := os.LookupEnv("FRONT_URL")
	if !found {
		frontURL = "http://localhost:3000"
	}

	// STEP 5-1: set up the database connection
	db, err := sql.Open("sqlite3", "db/mercari.sqlite3")
	if err != nil {
		slog.Error("failed to connect to database: ", "error", err)
		return 1
	}
	defer db.Close()

	// set up handlers
	itemRepo,err := NewItemRepository(db)
	if err != nil {
		slog.Error("failed to create item repository: ", "error", err)
		return 1
	}
	h := &Handlers{imgDirPath: s.ImageDirPath, itemRepo: itemRepo}

	// set up routes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.Hello)
	mux.HandleFunc("POST /items", h.AddItem)
	mux.HandleFunc("GET /items", h.GetItems)
	mux.HandleFunc("GET /items/{id}", h.GetItemById)
	mux.HandleFunc("GET /images/{filename}", h.GetImage)
	mux.HandleFunc("GET /search", h.Search)

	// start the server
	slog.Info("http server started on", "port", s.Port)
	err = http.ListenAndServe(":"+s.Port, simpleCORSMiddleware(simpleLoggerMiddleware(mux), frontURL, []string{"GET", "HEAD", "POST", "OPTIONS"}))
	if err != nil {
		slog.Error("failed to start server: ", "error", err)
		return 1
	}

	return 0
}

type Handlers struct {
	// imgDirPath is the path to the directory storing images.
	imgDirPath string
	itemRepo   ItemRepository
}

type HelloResponse struct {
	Message string `json:"message"`
}

// Hello is a handler to return a Hello, world! message for GET / .
func (s *Handlers) Hello(w http.ResponseWriter, r *http.Request) {
	resp := HelloResponse{Message: "Hello, world!"}
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type AddItemRequest struct {
	Name string `form:"name"`
	Category string `form:"category"` // STEP 4-2: add a category field
	Image []byte `form:"image"` // STEP 4-4: add an image field
}

type AddItemResponse struct {
	Message string `json:"message"`
}

// parseAddItemRequest parses and validates the request to add an item.
func parseAddItemRequest(r *http.Request) (*AddItemRequest, error) {
	var req = &AddItemRequest{}

	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		err := r.ParseMultipartForm(32 << 20) // 32MB
		if err != nil {
			return nil, fmt.Errorf("failed to parse multipart form: %w", err)
		}
	
		req.Name = r.FormValue("name")
		req.Category = r.FormValue("category")

		file, header, err := r.FormFile("image")
		if err != nil {
			if !errors.Is(err, http.ErrMissingFile) {
				return nil, fmt.Errorf("failed to get image file: %w", err)
			}
		} else {
			defer file.Close()

			if !strings.HasSuffix(strings.ToLower(header.Filename), ".jpg") && !strings.HasSuffix(strings.ToLower(header.Filename), ".jpeg") {
				return nil, errors.New("only .jpg or .jpeg files are allowed")
			}

			imageData, err := io.ReadAll(file)
			if err != nil {
				return nil, fmt.Errorf("failed to read image data: %w", err)
			}
			if len(imageData) == 0 {
				return nil, errors.New("image data is empty")
			}

			req.Image = imageData
		}
	
	} else {
		err := r.ParseForm()
		if err != nil {
			return nil, fmt.Errorf("failed to parse form: %w", err)
		}

		req.Name = r.FormValue("name")
		req.Category = r.FormValue("category")
	}

	// validate the request
	if req.Name == "" {
		return nil, errors.New("name is required")
	}
	if req.Category == "" {
		return nil, errors.New("category is required")
	}

	return req, nil
}

// AddItem is a handler to add a new item for POST /items .
func (s *Handlers) AddItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req, err := parseAddItemRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fileName, err := s.storeImage(req.Image)
	if err != nil {
		slog.Error("failed to store image: ", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	item := &Item{
		Name: req.Name,
		// STEP 4-2: add a category field
		Category: req.Category,
		// STEP 4-4: add an image field
		ImageName: fileName,
	}
	message := fmt.Sprintf("item received: %s (category: %s)", item.Name, item.Category)
	slog.Info(message)

	// STEP 4-2: add an implementation to store an item
	err = s.itemRepo.Insert(ctx, item)
	if err != nil {
		slog.Error("failed to store item: ", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := AddItemResponse{Message: message}
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// storeImage stores an image and returns the file path and an error if any.
// this method calculates the hash sum of the image as a file name to avoid the duplication of a same file
// and stores it in the image directory.
func (s *Handlers) storeImage(image []byte) (filePath string, err error) {
	// STEP 4-4: add an implementation to store an image
	// TODO:
	// - calc hash sum
	// - build image file path
	// - check if the image already exists
	// - store image
	// - return the image file path

	hash := sha256.Sum256(image)
	fileName := fmt.Sprintf("%x.jpg", hash)
	filePath = filepath.Join(s.imgDirPath, fileName)

	if _, err := os.Stat(filePath); err == nil {
		return filePath, nil
	}

	err = os.WriteFile(filePath, image, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write image file: %w", err)
	}

	return filePath, nil
}

type GetImageRequest struct {
	FileName string // path value
}

// parseGetImageRequest parses and validates the request to get an image.
func parseGetImageRequest(r *http.Request) (*GetImageRequest, error) {
	req := &GetImageRequest{
		FileName: r.PathValue("filename"), // from path parameter
	}

	// validate the request
	if req.FileName == "" {
		return nil, errors.New("filename is required")
	}

	return req, nil
}

// GetImage is a handler to return an image for GET /images/{filename} .
// If the specified image is not found, it returns the default image.
func (s *Handlers) GetImage(w http.ResponseWriter, r *http.Request) {
	req, err := parseGetImageRequest(r)
	if err != nil {
		slog.Warn("failed to parse get image request: ", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	imgPath, err := s.buildImagePath(req.FileName)
	if err != nil {
		if !errors.Is(err, errImageNotFound) {
			slog.Warn("failed to build image path: ", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// when the image is not found, it returns the default image without an error.
		slog.Debug("image not found", "filename", imgPath)
		imgPath = filepath.Join(s.imgDirPath, "default.jpg")
	}

	slog.Info("returned image", "path", imgPath)
	http.ServeFile(w, r, imgPath)
}

// buildImagePath builds the image path and validates it.
func (s *Handlers) buildImagePath(imageFileName string) (string, error) {
	imgPath := filepath.Join(s.imgDirPath, filepath.Clean(imageFileName))

	// to prevent directory traversal attacks
	rel, err := filepath.Rel(s.imgDirPath, imgPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("invalid image path: %s", imgPath)
	}

	// validate the image suffix
	if !strings.HasSuffix(imgPath, ".jpg") && !strings.HasSuffix(imgPath, ".jpeg") {
		return "", fmt.Errorf("image path does not end with .jpg or .jpeg: %s", imgPath)
	}

	// check if the image exists
	_, err = os.Stat(imgPath)
	if err != nil {
		return imgPath, errImageNotFound
	}

	return imgPath, nil
}


func (s *Handlers) GetItems(w http.ResponseWriter, r *http.Request) {
	items, err := s.itemRepo.GetAll(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := struct {
		Items []struct {
			ID       int    `json:"id"`
			Name     string `json:"name"`
			Category string `json:"category"`
			Image    string `json:"image_name"`
		} `json:"items"`
	}{}

	for _, item := range items {
		response.Items = append(response.Items, struct {
			ID       int    `json:"id"`
			Name     string `json:"name"`
			Category string `json:"category"`
			Image    string `json:"image_name"`
		}{
			ID:       item.ID,
			Name:     item.Name,
			Category: item.Category,
			Image:    item.ImageName,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}


type GetItemByIdRequest struct {
	Id string
}

func parseGetItemByIdRequest(r *http.Request) (*GetItemByIdRequest, error) {
	req := &GetItemByIdRequest{
		Id: r.PathValue("item_id"),
	}

	if req.Id == "" {
		return nil, errors.New("id is required")
	}

	return req, nil
}

func (s *Handlers) GetItemById(w http.ResponseWriter, r *http.Request) {
	req, err := parseGetItemByIdRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	item, err := s.itemRepo.GetByID(r.Context(), req.Id)
	if err != nil {
		if errors.Is(err, errItemNotFound) {
			slog.Warn("item not exist: ", "error", err)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	jsonData, err := json.Marshal(item)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}


type GetItemByKeywordRequest struct {
	Keyword string
}

func parseGetItemByKeywordRequest(r *http.Request) (*GetItemByKeywordRequest, error) {
	req := &GetItemByKeywordRequest{
		// クエリパラメータを取得
		Keyword: r.URL.Query().Get("keyword"),
	}

	if req.Keyword == "" {
		return nil, errors.New("keyword is required")
	}

	return req, nil
}

func (s *Handlers) Search(w http.ResponseWriter, r *http.Request) {
	req, err := parseGetItemByKeywordRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	items, err := s.itemRepo.Search(r.Context(), req.Keyword)

	if err != nil {
		if errors.Is(err, errItemNotFound) {
			slog.Warn("item not exist: ", "error", err)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	if items == nil {
		items = []Item{}
	}

	jsonData, err := json.Marshal(items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}