package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const uploadDir = "./uploads"
const logFileName = "server.log"
const version = "1.1.0"

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

func isTextHeuristic(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 8000)
	n, _ := f.Read(buf)
	buf = buf[:n]

	if n == 0 {
		return true
	}

	var nonPrintable int
	for _, b := range buf {
		if b == 0 {
			return false
		}
		if b < 32 && b != 9 && b != 10 && b != 13 {
			nonPrintable++
		}
	}

	return float64(nonPrintable)/float64(len(buf)) < 0.3
}

// Check file and size
func shouldForceDownload(path string, size int64) bool {
	f, err := os.Open(path)
	if err != nil {
		return true
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, _ := f.Read(buf)

	detectedMime := http.DetectContentType(buf[:n])
	extMime := mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))

	// allow inline
	if strings.HasPrefix(detectedMime, "image/") ||
		strings.HasPrefix(detectedMime, "video/") ||
		detectedMime == "application/pdf" {
		return false
	}

	// text (MIME or fallback)
	if strings.HasPrefix(detectedMime, "text/") || strings.HasPrefix(extMime, "text/") || isTextHeuristic(path) {
		if size > 10*1024 {
			return true
		}
		return false
	}

	return true
}

func withVersionHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Server-Version", version)
		next.ServeHTTP(w, r)
	})
}

// Handle file uploads
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errMsg := "Only POST method allowed"
		http.Error(w, errMsg, http.StatusMethodNotAllowed)
		logRequest(r, http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form with a reasonable max memory (32 MB)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		errMsg := fmt.Sprintf("Failed to parse form: %v", err)
		http.Error(w, errMsg, http.StatusBadRequest)
		logRequest(r, http.StatusBadRequest)
		return
	}

	// Get the uploaded file
	file, header, err := r.FormFile("file")
	if err != nil {
		errMsg := fmt.Sprintf("Failed to get file from request: %v", err)
		http.Error(w, errMsg, http.StatusBadRequest)
		logRequest(r, http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Check if file is empty
	if header.Size == 0 {
		errMsg := "Uploaded file is empty"
		http.Error(w, errMsg, http.StatusBadRequest)
		logRequest(r, http.StatusBadRequest)
		return
	}

	// Generate a random path
	randomPath := randomString(10)
	fileName := header.Filename

	// Validate filename to prevent path traversal
	if strings.Contains(fileName, "..") || strings.Contains(fileName, "/") || strings.Contains(fileName, "\\") {
		errMsg := "Invalid filename: contains path traversal characters"
		http.Error(w, errMsg, http.StatusBadRequest)
		logRequest(r, http.StatusBadRequest)
		return
	}

	dirPath := filepath.Join(uploadDir, randomPath)

	// Create directory
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		errMsg := fmt.Sprintf("Failed to create directory '%s': %v", dirPath, err)
		http.Error(w, errMsg, http.StatusInternalServerError)
		logRequest(r, http.StatusInternalServerError)
		return
	}

	// Full file path
	filePath := filepath.Join(dirPath, fileName)

	// Check if file already exists
	if _, err := os.Stat(filePath); err == nil {
		errMsg := fmt.Sprintf("File '%s' already exists in the target location", fileName)
		http.Error(w, errMsg, http.StatusConflict)
		logRequest(r, http.StatusConflict)
		return
	}

	// Save the file
	outFile, err := os.Create(filePath)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create file '%s': %v", filePath, err)
		http.Error(w, errMsg, http.StatusInternalServerError)
		logRequest(r, http.StatusInternalServerError)
		return
	}
	defer outFile.Close()

	// Copy file contents and check for errors
	bytesWritten, err := io.Copy(outFile, file)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to write file contents: %v", err)
		http.Error(w, errMsg, http.StatusInternalServerError)
		logRequest(r, http.StatusInternalServerError)
		// Try to clean up the partially written file
		os.Remove(filePath)
		return
	}

	// Verify that all bytes were written
	if bytesWritten != header.Size {
		errMsg := fmt.Sprintf("Incomplete upload: wrote %d bytes but expected %d bytes", bytesWritten, header.Size)
		http.Error(w, errMsg, http.StatusInternalServerError)
		logRequest(r, http.StatusInternalServerError)
		// Clean up the incomplete file
		os.Remove(filePath)
		return
	}

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

	fileInfo, err := os.Stat(filePath)
	if err == nil {
		if shouldForceDownload(filePath, fileInfo.Size()) {
			w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(fileName))
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("X-Content-Type-Options", "nosniff")
		}
	}

	http.ServeFile(w, r, filePath)
	logRequest(r, http.StatusOK)
}

// displayHelp shows usage information
func displayHelp() {
	fmt.Printf(`Usage: %s [options]

A simple file upload and download server.

Options:
  -v, --version     Show version information and exit
  -h, --help        Show this help message and exit
  -p, --port PORT   Port to listen on (default: 5555)
  -i, --ipaddr IP   IP address to bind to (default: 0.0.0.0, supports both IPv4 and IPv6)

Examples:
  %s                      # Start server on default 0.0.0.0:5555
  %s -p 8080              # Start server on port 8080
  %s -i 127.0.0.1         # Start server on localhost only
  %s -i ::1 -p 8080       # Start server on IPv6 localhost port 8080
  %s -i 0.0.0.0 -p 9000   # Start server on all interfaces port 9000

Note: The server creates an 'uploads' directory for storing files and logs to server.log
`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

func main() {
	// Define command line flags
	var (
		showVersion bool
		showHelp    bool
		port        string
		ipAddr      string
	)

	// Custom usage message
	flag.Usage = displayHelp

	// Define flags with both short and long forms
	flag.BoolVar(&showVersion, "v", false, "Show version")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&showHelp, "h", false, "Show help")
	flag.BoolVar(&showHelp, "help", false, "Show help")
	flag.StringVar(&port, "p", "5555", "Port to listen on")
	flag.StringVar(&port, "port", "5555", "Port to listen on")
	flag.StringVar(&ipAddr, "i", "0.0.0.0", "IP address to bind to")
	flag.StringVar(&ipAddr, "ipaddr", "0.0.0.0", "IP address to bind to")

	// Parse command line arguments
	flag.Parse()

	// Check for version flag
	if showVersion {
		fmt.Printf("File Upload Server version %s\n", version)
		os.Exit(0)
	}

	// Check for help flag
	if showHelp {
		displayHelp()
		os.Exit(0)
	}

	// Check for invalid arguments (unexpected positional arguments)
	if flag.NArg() > 0 {
		fmt.Printf("Error: Unknown argument(s): %v\n", flag.Args())
		fmt.Println("Use -h or --help for usage information")
		os.Exit(1)
	}

	// Validate port (basic validation)
	if port == "" {
		fmt.Println("Error: Port cannot be empty")
		os.Exit(1)
	}

	// Validate IP address format (optional check)
	if ipAddr != "" {
		// Try to parse the IP address to validate format (works for both IPv4 and IPv6)
		if net.ParseIP(ipAddr) == nil && ipAddr != "0.0.0.0" && ipAddr != "::" {
			fmt.Printf("Warning: '%s' doesn't appear to be a valid IP address, but will attempt to bind anyway\n", ipAddr)
		}
	}

	// Create uploads directory if it doesn't exist
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		log.Fatalf("Failed to create uploads directory: %v", err)
	}

	// Setup HTTP handlers
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/f/", fileHandler)

	// Determine the listen address (supports IPv6)
	listenAddr := net.JoinHostPort(ipAddr, port)

	// Start the server
	fmt.Printf("Server version %s started at http://%s\n", version, listenAddr)
	fmt.Printf("Upload endpoint: http://%s/upload\n", listenAddr)
	fmt.Printf("Files will be accessible via: http://%s/f/{random_path}/{filename}\n", listenAddr)
	log.Printf("Server started on %s", listenAddr)

	// ListenAndServe supports both IPv4 and IPv6 natively
	if err := http.ListenAndServe(listenAddr, withVersionHeader(http.DefaultServeMux)); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
