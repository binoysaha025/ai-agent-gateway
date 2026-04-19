package tools

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

type SearchInput struct {
	Query string `json:"query"`
}

func SearchKnowledge(input json.RawMessage, db *sql.DB) string {
	var params SearchInput
	if err := json.Unmarshal(input, &params); err != nil {
		return "error parsing input"
	}

	// embed the query
	embedding, err := GetEmbedding(params.Query)
	if err != nil {
		return fmt.Sprintf("error getting embedding: %s", err.Error())
	}

	// convert embedding to postgres vector string format
	dims := make([]string, len(embedding))
	for i, v := range embedding {
		dims[i] = fmt.Sprintf("%f", v)
	}
	vectorStr := "[" + strings.Join(dims, ",") + "]"

	// cosine similarity search in postgres - top 3 results
	rows, err := db.Query(`
		SELECT content, metadata, 1 - (embedding <=> $1::vector) AS similarity
		FROM documents
		ORDER BY embedding <=> $1::vector
		LIMIT 3
	`, vectorStr)
	if err != nil {
		return fmt.Sprintf("error searching database: %s", err.Error())
	}
	defer rows.Close()

	results := []string{}
	for rows.Next() {
		var content, metadata string
		var similarity float64
		if err := rows.Scan(&content, &metadata, &similarity); err != nil {
			continue
		}
		results = append(results, fmt.Sprintf("[similarity: %.2f] %s", similarity, content))
	}

	return "Relevant context from knowledge base:\n" + strings.Join(results, "\n\n")

}