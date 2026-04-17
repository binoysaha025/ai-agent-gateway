package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

// calculator tool

type CalcInput struct {
	Expression string `json:"expression"`
}

func Calculate(input json.RawMessage) string {
	var params CalcInput
	if err := json.Unmarshal(input, &params); err != nil{
		return "error parsing input"
	}
	// simple eval approach
	result, err := evalExpression(params.Expression)
	if err != nil {
		return fmt.Sprintf("error: %s", err.Error())
	}
	return fmt.Sprintf("%g", result)
}

func evalExpression(expr string)(float64, error) {
	// santize - only allows numbers and basic operators
	for _, ch := range expr {
		if !strings.ContainsRune("0123456789+-*/(). ", ch) {
			return 0, fmt.Errorf("invalid character in expression")
		}
	}

	// use Go's net/http to hit a free math eval API
	url := fmt.Sprintf("https://api.mathjs.org/v4/?expr=%s", strings.ReplaceAll(expr, " ", "+"))
	resp, err := http.Get(url)
	if err != nil{
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var result float64
	_, err = fmt.Sscanf(string(body), "%g", &result)
	if err != nil {
		return 0, fmt.Errorf("could not parse result: %s", string(body))
	}
	return result, nil
}

// URL Summarizer tool
type SummarizeInput struct {
	URL string `json:"url"`
}

func SummarizeURL(input json.RawMessage) string {
	var params SummarizeInput
	if err := json.Unmarshal(input, &params); err != nil {
		return "error parsing input"
	}

	// fetch the page
	resp, err := http.Get(params.URL)
	if err != nil {
		return fmt.Sprintf("error extracting text: %s", err.Error())
	}
	defer resp.Body.Close()

	// extract text from HTML
	text, err := extractText(resp.Body)
	if err != nil {
		return fmt.Sprintf("error extracting text: %s", err.Error())
	}

	// truncate to 3000 characters so context is not too long
	if len(text) > 3000 {
		text = text[:3000] + "..."
	}
	return text
}

func extractText(r io.Reader) (string, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		// skip script and style tags
		if n.Type == html.ElementNode {
			if n.Data == "script" || n.Data == "style" {
				return
			}
		}
		//grab text nodes
		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				buf.WriteString (text + " ")
			}
		}
		// recurse into children
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return buf.String(), nil
}

// news fetcher tool
type NewsInput struct {
	Topic string `json:"topic"`
}

type HNStory struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Score int    `json:"score"`
}

func FetchNews(input json.RawMessage) string {
	var params NewsInput
	if err := json.Unmarshal(input, &params); err != nil {
		return "error parsing input"
	}
	// fetch top 20 HN stories matching the topic
	resp, err := http.Get("https://hacker-news.firebaseio.com/v0/topstories.json")
	if err != nil {
		return fmt.Sprintf("error fecthing news : %s", err.Error())
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("error reading response: %s", err.Error())
	}

	var ids []int
	if err := json.Unmarshal(body, &ids); err != nil {
		return fmt.Sprintf("error parsing response: %s", err.Error())
	}

	// fetch details for top 20 stories
	stories := []string{}
	for i := 0; i < 20 && i < len(ids); i++ {
		storyURL := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", ids[i])
		storyResp, err := http.Get(storyURL)
		if err != nil {
			continue
		}
		defer storyResp.Body.Close()

		storyBody, err := io.ReadAll(storyResp.Body)
		if err != nil {
			continue
		}

		var story HNStory
		if err := json.Unmarshal(storyBody, &story); err != nil {
			continue
		}

		// client side topic filer
		if params.Topic != "" && !strings.Contains(
			strings.ToLower(story.Title),
			strings.ToLower(params.Topic),
		) {
			continue
		}

		stories = append(stories, fmt.Sprintf("- %s (%d points) %s", story.Title, story.Score, story.URL))

		// stop after 5 relevant stories
		if len(stories) == 5 {
			break
		}
	}

	if len(stories) == 0 {
		return  fmt.Sprintf("no relevant news found for topic: %s", params.Topic)
	}

	return "Top Hacker News Stories:\n" + strings.Join(stories, "\n")
}