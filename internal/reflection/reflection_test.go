package reflection

import (
	"strings"
	"testing"
)

func TestDefaultQuestionsReturns7(t *testing.T) {
	questions := DefaultQuestions()
	if len(questions) != 7 {
		t.Errorf("DefaultQuestions() returned %d questions, want 7", len(questions))
	}
}

func TestDefaultQuestionsIncludesTestDiscovery(t *testing.T) {
	questions := DefaultQuestions()
	found := false
	for _, q := range questions {
		if strings.Contains(q.Question, "new test scenarios") {
			found = true
			break
		}
	}
	if !found {
		t.Error("DefaultQuestions() should include a test-discovery question about new test scenarios")
	}
}

func TestDefaultQuestionsSequentialIDs(t *testing.T) {
	questions := DefaultQuestions()
	for i, q := range questions {
		expectedID := i + 1
		if q.ID != expectedID {
			t.Errorf("question[%d].ID = %d, want %d", i, q.ID, expectedID)
		}
	}
}

func TestDefaultQuestionsStartUnanswered(t *testing.T) {
	questions := DefaultQuestions()
	for _, q := range questions {
		if q.Answer != "" {
			t.Errorf("question %d should start unanswered, got %q", q.ID, q.Answer)
		}
	}
}

func TestDefaultQuestionsHaveText(t *testing.T) {
	questions := DefaultQuestions()
	for _, q := range questions {
		if q.Question == "" {
			t.Errorf("question %d has empty text", q.ID)
		}
	}
}

func TestValidateAnswerRejectsTooFew(t *testing.T) {
	tests := []struct {
		name   string
		answer string
	}{
		{"empty", ""},
		{"one word", "no"},
		{"two words", "not really"},
		{"three words", "tests are fine"},
		{"four words", "tests are all fine"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateAnswer(tt.answer); err == nil {
				t.Errorf("ValidateAnswer(%q) should return error for <5 words", tt.answer)
			}
		})
	}
}

func TestValidateAnswerAcceptsEnough(t *testing.T) {
	tests := []struct {
		name   string
		answer string
	}{
		{"exactly 5 words", "tests are already very clear"},
		{"more than 5 words", "I renamed test functions to better describe their behavior"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateAnswer(tt.answer); err != nil {
				t.Errorf("ValidateAnswer(%q) returned unexpected error: %v", tt.answer, err)
			}
		})
	}
}
