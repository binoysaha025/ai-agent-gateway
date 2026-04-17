package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/binoysaha025/ai-agent-gateway/models"
)

type Handler struct {
	DB *sql.DB
}

func NewHandler (db *sql.DB) *Handler {
	return &Handler{DB: db}
}

func generateKey() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (h *Handler) CreateAPIKey(c *gin.Context) {
	var body struct {
		Name string `json:"name"`
		Plan string `json:"plan"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if body.Plan == "" {
		body.Plan = "free"
	}

	key, err := generateKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate API key"})
		return
	}

	err = models.InsertAPIKey(h.DB, key, body.Name, body.Plan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save API key"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"key": key,
		"name": body.Name,
		"plan": body.Plan,
	})
}