// Example CLI command to list routes (like Laravel's php artisan route:list)
//
// Usage:
//
//	go run examples/routes-cli/main.go
//	go run examples/routes-cli/main.go --json
//	go run examples/routes-cli/main.go --compact
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

var (
	jsonOutput    = flag.Bool("json", false, "Output routes as JSON")
	compactOutput = flag.Bool("compact", false, "Output routes in compact format")
)

func main() {
	flag.Parse()

	// Create router and register routes
	// In a real app, this would be your SetupRoutes() function
	router := setupRoutes()

	// Get all routes
	routes := router.Routes()

	// Format output based on flags
	var err error
	if *jsonOutput {
		err = teapot.FormatRoutesJSON(os.Stdout, routes)
	} else if *compactOutput {
		err = teapot.FormatRoutesCompact(os.Stdout, routes)
	} else {
		err = teapot.FormatRoutesTable(os.Stdout, routes)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting routes: %v\n", err)
		os.Exit(1)
	}
}

// setupRoutes demonstrates a typical route registration
// In your app, this would be in your server setup code
func setupRoutes() *teapot.Router {
	router := teapot.New()

	// Public routes
	router.GET("/", homeHandler).Name("home")
	router.GET("/about", aboutHandler).Name("about")

	// API routes (basic RESTful)
	router.GET("/api/users", listUsers).Name("api.users.index")
	router.POST("/api/users", createUser).Name("api.users.store")
	router.GET("/api/users/{id}", showUser).Name("api.users.show")
	router.PUT("/api/users/{id}", updateUser).Name("api.users.update")
	router.DELETE("/api/users/{id}", deleteUser).Name("api.users.destroy")

	router.GET("/api/posts", listPosts).Name("api.posts.index")
	router.POST("/api/posts", createPost).Name("api.posts.store")
	router.GET("/api/posts/{id}", showPost).Name("api.posts.show")
	router.PUT("/api/posts/{id}", updatePost).Name("api.posts.update")

	// S3 API routes
	router.QueryGET("/", listBuckets).Query("list-type").Name("s3.buckets.list").Action("s3:ListBuckets")
	router.PUT("/{bucket}", createBucket).Name("s3.buckets.create").Action("s3:CreateBucket")
	router.DELETE("/{bucket}", deleteBucket).Name("s3.buckets.delete").Action("s3:DeleteBucket")

	router.QueryGET("/{bucket}", listObjects).Query("list-type").Name("s3.objects.list").Action("s3:ListObjects")
	router.GET("/{bucket}/{key:.*}", getObject).Name("s3.objects.get").Action("s3:GetObject")
	router.PUT("/{bucket}/{key:.*}", putObject).Name("s3.objects.put").Action("s3:PutObject")
	router.DELETE("/{bucket}/{key:.*}", deleteObject).Name("s3.objects.delete").Action("s3:DeleteObject")

	// Admin routes
	router.GET("/admin", adminDashboard).Name("admin.dashboard")
	router.GET("/admin/users", adminUsers).Name("admin.users")

	// Debug route (conditionally registered)
	if isDebug() {
		router.RegisterDebugRoute("/.internal/routes", "debug.routes")
	}

	return router
}

// Mock handlers
func homeHandler(w http.ResponseWriter, r *http.Request)    {}
func aboutHandler(w http.ResponseWriter, r *http.Request)   {}
func listUsers(w http.ResponseWriter, r *http.Request)      {}
func createUser(w http.ResponseWriter, r *http.Request)     {}
func showUser(w http.ResponseWriter, r *http.Request)       {}
func updateUser(w http.ResponseWriter, r *http.Request)     {}
func deleteUser(w http.ResponseWriter, r *http.Request)     {}
func listPosts(w http.ResponseWriter, r *http.Request)      {}
func createPost(w http.ResponseWriter, r *http.Request)     {}
func showPost(w http.ResponseWriter, r *http.Request)       {}
func updatePost(w http.ResponseWriter, r *http.Request)     {}
func listBuckets(w http.ResponseWriter, r *http.Request)    {}
func createBucket(w http.ResponseWriter, r *http.Request)   {}
func deleteBucket(w http.ResponseWriter, r *http.Request)   {}
func listObjects(w http.ResponseWriter, r *http.Request)    {}
func getObject(w http.ResponseWriter, r *http.Request)      {}
func putObject(w http.ResponseWriter, r *http.Request)      {}
func deleteObject(w http.ResponseWriter, r *http.Request)   {}
func adminDashboard(w http.ResponseWriter, r *http.Request) {}
func adminUsers(w http.ResponseWriter, r *http.Request)     {}
func isDebug() bool                                         { return os.Getenv("DEBUG") == "true" }
