package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const uploadDir = "./uploads"
const logFileName = "server.log"

// Generate a random string (10 characters)
func randomString(n int) string {
	bytes := make([]byte, n)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

// Log requests to both console and log file
func logRequest(r *http.Request, status int) {
	logEntry := fmt.Sprintf("%s - %s %s %s - %d", time.Now().Format("2006-01-02 15:04:05"), r.RemoteAddr, r.Method, r.URL.Path, status)
	fmt.Println(logEntry)

	logFile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		return
	}
	defer logFile.Close()

	logger := log.New(logFile, "", 0)
	logger.Println(logEntry)
}

// Handle file uploads
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method allowed", http.StatusMethodNotAllowed)
		logRequest(r, http.StatusMethodNotAllowed)
		return
	}

	// Get the uploaded file
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file", http.StatusBadRequest)
		logRequest(r, http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Generate a random path
	randomPath := randomString(10)
	fileName := header.Filename
	dirPath := filepath.Join(uploadDir, randomPath)

	// Create directory
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		http.Error(w, "Failed to create directory", http.StatusInternalServerError)
		logRequest(r, http.StatusInternalServerError)
		return
	}

	// Full file path
	filePath := filepath.Join(dirPath, fileName)

	// Save the file
	outFile, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		logRequest(r, http.StatusInternalServerError)
		return
	}
	defer outFile.Close()
	io.Copy(outFile, file)

	// Return file URL
	fileURL := fmt.Sprintf("http://%s/f/%s/%s", r.Host, randomPath, fileName)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fileURL))
	logRequest(r, http.StatusCreated)
}

// Handle file downloads
func fileHandler(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.NotFound(w, r)
		logRequest(r, http.StatusNotFound)
		return
	}

	randomPath := pathParts[2]
	fileName := pathParts[3]
	filePath := filepath.Join(uploadDir, randomPath, fileName)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.NotFound(w, r)
		logRequest(r, http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, filePath)
	logRequest(r, http.StatusOK)
}

func main() {
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/f/", fileHandler)

	port := "5555"
	fmt.Printf("Server started at http://0.0.0.0:%s\n", port)
	log.Println("Server started on port", port)
	http.ListenAndServe("0.0.0.0:"+port, nil)
}
