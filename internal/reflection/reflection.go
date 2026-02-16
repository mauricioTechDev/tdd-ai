package reflection

import (
	"fmt"
	"strings"

	"github.com/macosta/tdd-ai/internal/types"
)

// MinAnswerWords is the minimum number of words required for a valid reflection answer.
const MinAnswerWords = 5

// DefaultQuestions returns the 7 structured reflection questions with sequential IDs.
func DefaultQuestions() []types.ReflectionQuestion {
	return []types.ReflectionQuestion{
		{ID: 1, Question: "Can I make my test suite more expressive?"},
		{ID: 2, Question: "Does my test suite provide reliable feedback?"},
		{ID: 3, Question: "Are my tests isolated?"},
		{ID: 4, Question: "Can I reduce duplication in my test suite or implementation code?"},
		{ID: 5, Question: "Can I make my implementation code more descriptive?"},
		{ID: 6, Question: "Can I implement something more efficiently?"},
		{ID: 7, Question: "Should any new test scenarios be added to the test list based on what I learned?"},
	}
}

// ValidateAnswer checks that an answer has at least MinAnswerWords words.
func ValidateAnswer(answer string) error {
	words := len(strings.Fields(answer))
	if words < MinAnswerWords {
		return fmt.Errorf("answer must be at least %d words, got %d", MinAnswerWords, words)
	}
	return nil
}
