package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/binoysaha025/ai-agent-gateway/models"
	"fmt"
	"strings"
	"github.com/binoysaha025/ai-agent-gateway/tools"
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

func (h *Handler) EmbedDocument(c *gin.Context) {
	var body struct {
		Content string `json:"content"`
		Metadata string `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&body); err != nil || body.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content is required"})
		return
	}
	
	// auto chunk into 500 character chunks to preserve semantic integrity of vectors
	chunks := chunkText(body.Content, 500)

	stored := 0
	for _, chunk := range chunks {
		embedding, err := tools.GetEmbedding(chunk)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate embedding"})
			return
		}

		dims := make([]string, len(embedding))
		for i, v := range embedding {
			dims[i] = fmt.Sprintf("%f", v)
		}
		vectorStr := "[" + strings.Join(dims, ",") + "]"

		_, err = h.DB.Exec(
			`INSERT INTO documents (content, embedding, metadata) VALUES ($1, $2::vector, $3)`,
			chunk, vectorStr, body.Metadata,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store chunk"})
			return
		}
		stored++
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "document embedded and stored",
		"chunks":  stored,
	})
}

func chunkText(text string, maxChars int) []string {
	// split on sentence endings first
	sentences := strings.FieldsFunc(text, func(r rune) bool {
		return r == '.' || r == '!' || r == '?'
	})

	chunks := []string{}
	current := ""

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}
		// add period back
		sentence = sentence + "."

		if len(current)+len(sentence) > maxChars && current != "" {
			chunks = append(chunks, strings.TrimSpace(current))
			current = sentence
		} else {
			current += " " + sentence
		}
	}

	if strings.TrimSpace(current) != "" {
		chunks = append(chunks, strings.TrimSpace(current))
	}

	return chunks
}