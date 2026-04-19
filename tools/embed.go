package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"log"
)

type VoyageRequest struct {
	Input []string `json:"input"`
	Model string `json:"model"`
}

type VoyageResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

func GetEmbedding(text string) ([]float64, error) {
	apiKey := os.Getenv("VOYAGE_API_KEY")
	log.Printf("voyage api key: %s", apiKey)

	reqBody := VoyageRequest{
		Input: []string{text},
		Model: "voyage-3",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://api.voyageai.com/v1/embeddings", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	log.Printf("voyage response: %s", string(respBody))

	var voyageResp VoyageResponse
	if err := json.Unmarshal(respBody, &voyageResp); err != nil {
		return nil, err
	}

	if len(voyageResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return voyageResp.Data[0].Embedding, nil
}