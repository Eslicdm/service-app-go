package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"service-app-go/member-service/core/config"
	"service-app-go/member-service/core/entity"
	"service-app-go/member-service/core/exception"
	"service-app-go/member-service/member/controller"
	"service-app-go/member-service/member/repository"
	"service-app-go/member-service/member/service"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load()

	dsn := fmt.Sprintf("host=localhost user=%s password=%s dbname=%s "+
		"port=5435 sslmode=disable TimeZone=UTC",
		os.Getenv("MEMBER_DB_USERNAME"),
		os.Getenv("MEMBER_DB_PASSWORD"),
		os.Getenv("MEMBER_DB_NAME"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&entity.Member{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Dependency Injection
	memberRepo := repository.NewMemberRepository(db)
	memberService := service.NewMemberService(memberRepo)
	memberController := controller.NewMemberController(memberService)
	securityConfig := config.NewSecurityConfig("http://keycloak:8080/realms/service-app-realm")

	r := gin.Default()

	r.Use(exception.GlobalErrorHandler())

	// Health check
	r.GET("/actuator/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	api := r.Group("/api/v1/members")
	api.Use(securityConfig.AuthMiddleware())
	{
		api.GET("", memberController.GetAllMembersByManagerID)
		api.GET("/:memberId", memberController.GetMemberByID)
		api.POST("", memberController.CreateMember)
		api.PUT("/:memberId", memberController.UpdateMember)
		api.DELETE("/:memberId", memberController.DeleteMember)
	}

	if err := r.Run(":8090"); err != nil {
		log.Fatal(err)
	}
}
