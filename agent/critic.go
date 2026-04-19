package agent

import (
	"fmt"
	"strings"
	"sync"
)

type CriticResult struct {
	Role string
	Verdict string		// pass or fail
	Reason string
}

type EnsembleVerdict struct {
	Passed bool
	Score int 			// how many critics passed (out of 3)
	Feedback string		// combined feedback to feed into Claude for retry IF fail
}

func runCritic(prompt, response, role, instruction string, wg *sync.WaitGroup, results chan<- CriticResult) {
	defer wg.Done()

	criticPrompt := fmt.Sprintf(`You are a %s critic evaluating an AI response.

Original prompt: %s

AI Response: %s

%s

Respond in this exact format:
VERDICT: PASS or FAIL
REASON: one sentence explanation`, role, prompt, response, instruction)

	messages := []Message{
		{Role: "user", Content: criticPrompt},
	}

	resp, err := callClaude(messages, []Tool{})
	if err != nil {
		results <- CriticResult{Role: role, Verdict: "PASS", Reason: "critic unavailable"}
		return
	}

	raw := ""
	for _, block := range resp.Content {
		if block.Type == "text" {
			raw = block.Text
			break
		}
	}

	verdict := "PASS"
	reason := ""

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "VERDICT:") {
			verdict = strings.TrimSpace(strings.TrimPrefix(line, "VERDICT:"))
		}
		if strings.HasPrefix(line, "REASON:") {
			reason = strings.TrimSpace(strings.TrimPrefix(line, "REASON:"))
		}
	}

	results <- CriticResult{Role: role, Verdict: verdict, Reason: reason}
}


func EvaluateResponse(prompt, response string) EnsembleVerdict {
	critics := []struct {
		role        string
		instruction string
	}{
		{
			role: "Factuality",
			instruction: "Does this response contain any factual errors, hallucinations, or unsupported claims? Consider whether the stated facts are accurate.",
		},
		{
			role: "Completeness", 
			instruction: "Does this response fully address what was asked? Check if any important aspects of the question were ignored or left unanswered.",
		},
		{
			role: "Groundedness",
			instruction: "Is this response grounded in verifiable information? Check if claims are made without evidence or if the response makes up specific details.",
		},
	}

	results := make(chan CriticResult, len(critics))
	var wg sync.WaitGroup

	// launch all 3 critics in parallel
	for _, critic := range critics {
		wg.Add(1)
		go runCritic(prompt, response, critic.role, critic.instruction, &wg, results)
	}

	// wait for all critics to finish then close channel
	go func() {
		wg.Wait()
		close(results)
	}()

	// collect results
	passed := 0
	failed := 0
	failReasons := []string{}

	for result := range results {
		if result.Verdict == "PASS" {
			passed++
		} else {
			failed++
			failReasons = append(failReasons, fmt.Sprintf("%s critic: %s", result.Role, result.Reason))
		}
	}

	feedback := strings.Join(failReasons, ". ")

	return EnsembleVerdict{
		Passed:   passed >= 2, // majority vote — 2 out of 3
		Score:    passed,
		Feedback: feedback,
	}
}