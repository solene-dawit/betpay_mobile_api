package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/joho/godotenv"

	"betpay_mobile_api/database"
	"betpay_mobile_api/handlers"
	"betpay_mobile_api/middleware"
	"betpay_mobile_api/services"
)

// =====================================
// HEALTH CHECK
// =====================================

func healthHandler(w http.ResponseWriter, r *http.Request) {

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	response := map[string]string{
		"message": "BetPay backend is running!",
		"status":  "ok",
	}

	json.NewEncoder(w).Encode(response)
}

// =====================================
// MAIN
// =====================================

func main() {

	// ---------------------------------
	// Load .env
	// ---------------------------------

	err := godotenv.Load()

	if err != nil {
		log.Println("Warning: .env file not found")
	}

	// ---------------------------------
	// Connect to PostgreSQL
	// ---------------------------------

	db, err := database.Connect()

	if err != nil {
		log.Fatal(
			"Database connection failed:",
			err,
		)
	}

	defer db.Close()

	// ---------------------------------
	// Test database
	// ---------------------------------

	var result int

	err = db.QueryRow(
		context.Background(),
		"SELECT 1",
	).Scan(&result)

	if err != nil {
		log.Fatal(
			"Database test failed:",
			err,
		)
	}

	log.Println(
		"PostgreSQL connected successfully!",
	)

	log.Println(
		"Database test result:",
		result,
	)

	// =================================
	// INITIALIZE SERVICES
	// =================================

	starPayService := services.NewStarPayService()

	// =================================
	// INITIALIZE HANDLERS
	// =================================

	starPayHandler := handlers.NewStarPayHandler(
		starPayService,
	)

	loginHandler := handlers.NewLoginHandler(db)

	paymentHandler := handlers.NewPaymentHandler(db)

	meHandler := handlers.NewMeHandler(db)
	socialHandler := handlers.NewSocialHandler(db)
	postHandler := handlers.NewPostHandler(db)

	// =================================
	// ROUTES
	// =================================

	// ---------------------------------
// Social / Community
// ---------------------------------

	http.HandleFunc(
		"POST /api/groups",
		socialHandler.CreateGroup,
	)

	http.HandleFunc(
		"GET /api/groups",
		socialHandler.GetGroups,
	)

	http.HandleFunc(
		"POST /api/groups/{groupId}/join",
		socialHandler.JoinGroup,
	)

	// ---------------------------------
// Social Posts
// ---------------------------------

	http.HandleFunc(
		"POST /api/groups/{groupId}/posts",
		postHandler.CreatePost,
	)
	http.HandleFunc(
	"GET /api/groups/{groupId}/posts",
	postHandler.GetGroupPosts,
	)
	http.HandleFunc(
		"POST /api/posts/{postId}/like",
		postHandler.LikePost,
	)
	http.HandleFunc(
	"POST /api/posts/{postId}/comments",
	postHandler.CreateComment,
   )
   http.HandleFunc(
	"GET /api/posts/{postId}/comments",
	postHandler.GetPostComments,
   )
   http.HandleFunc(
	"DELETE /api/posts/{postId}/like",
	postHandler.UnlikePost,
   )
   http.HandleFunc(
	"GET /api/posts/{postId}/likes",
	postHandler.GetLikeCount,
   )
   http.HandleFunc(
	"GET /api/posts/{postId}/like-status",
	postHandler.CheckLike,
	)
	http.HandleFunc(
		"DELETE /api/groups/{groupId}/leave",
		socialHandler.LeaveGroup,
	)
	http.HandleFunc(
	"GET /api/groups/{groupId}/membership",
	socialHandler.CheckMembership,
	)
	http.HandleFunc(
		"GET /api/groups/{groupId}/members/count",
		socialHandler.GetMemberCount,
	)
	http.HandleFunc(
	"GET /api/groups/{groupId}/members",
	socialHandler.GetMembers,
	)
	http.HandleFunc(
		"DELETE /api/comments/{commentId}",
		postHandler.DeleteComment,
	)
	http.HandleFunc(
		"DELETE /api/posts/{postId}",
		postHandler.DeletePost,
	)
	http.HandleFunc(
		"PUT /api/posts/{postId}",
		postHandler.UpdatePost,
	)
	http.HandleFunc(
		"PUT /api/comments/{commentId}",
		postHandler.UpdateComment,
	)
	http.HandleFunc(
	"GET /api/users/{userId}/groups",
	socialHandler.GetUserGroups,
    )

	http.HandleFunc(
		"/api/health",
		healthHandler,
	)
	http.HandleFunc(
	"GET /api/users/{userId}/posts",
	postHandler.GetUserPosts,
	)
	http.HandleFunc(
		"GET /api/users/{userId}/comments",
		postHandler.GetUserComments,
	)
	// ---------------------------------
	// User registration
	// ---------------------------------

	http.HandleFunc(
		"/api/register",
		handlers.RegisterHandler(db),
	)

	// ---------------------------------
	// Login
	// ---------------------------------

	http.HandleFunc(
		"/api/login",
		loginHandler.Login,
	)

	// ---------------------------------
	// Current logged-in user
	// Protected by JWT middleware
	// ---------------------------------

	http.Handle(
		"/api/me",
		middleware.AuthMiddleware(
			http.HandlerFunc(meHandler.GetMe),
		),
	)

	// ---------------------------------
	// StarPay payment creation
	// ---------------------------------

	http.HandleFunc(
		"/api/payments/starpay",
		starPayHandler.CreatePayment,
	)

	// ---------------------------------
	// Save payment
	// ---------------------------------

	http.HandleFunc(
		"/api/payments",
		paymentHandler.SavePayment,
	)

	// =================================
	// SERVER INFORMATION
	// =================================

	log.Println(
		"BetPay Go backend is running on:",
	)

	log.Println(
		"http://localhost:8080",
	)

	log.Println(
		"Health check:",
	)

	log.Println(
		"http://localhost:8080/api/health",
	)

	log.Println(
		"Registration API:",
	)

	log.Println(
		"POST http://localhost:8080/api/register",
	)

	log.Println(
		"Login API:",
	)

	log.Println(
		"POST http://localhost:8080/api/login",
	)

	log.Println(
		"Current User API:",
	)

	log.Println(
		"GET http://localhost:8080/api/me",
	)

	log.Println(
		"StarPay Payment API:",
	)

	log.Println(
		"POST http://localhost:8080/api/payments/starpay",
	)

	log.Println(
		"Save Payment API:",
	)

	log.Println(
		"POST http://localhost:8080/api/payments",
	)

	// =================================
	// START SERVER
	// =================================

	err = http.ListenAndServe(
		":8080",
		nil,
	)

	if err != nil {
		log.Fatal(err)
	}
}