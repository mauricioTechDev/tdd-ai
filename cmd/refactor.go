package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/macosta/tdd-ai/internal/formatter"
	"github.com/macosta/tdd-ai/internal/reflection"
	"github.com/macosta/tdd-ai/internal/session"
	"github.com/macosta/tdd-ai/internal/types"
	"github.com/spf13/cobra"
)

var refactorCmd = &cobra.Command{
	Use:   "refactor",
	Short: "Show refactor reflection status",
	Long:  "Show a brief summary of reflection question progress during the refactor phase.",
	Example: `  tdd-ai refactor
  tdd-ai refactor status
  tdd-ai refactor reflect 1 --answer "Tests are already descriptive and clear"`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		dir := getWorkDir()
		s, err := session.LoadOrFail(dir)
		if err != nil {
			return err
		}

		if s.Phase != types.PhaseRefactor {
			return fmt.Errorf("not in refactor phase (current: %s)", s.Phase)
		}

		answered := 0
		for _, r := range s.Reflections {
			if r.Answer != "" {
				answered++
			}
		}
		total := len(s.Reflections)

		if total == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No reflection questions loaded.")
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Reflections: %d/%d answered\n", answered, total)
		if answered < total {
			fmt.Fprintln(cmd.OutOrStdout(), "Run 'tdd-ai refactor status' to see all questions")
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "All reflection questions answered. Ready to advance.")
		}
		return nil
	},
}

var reflectAnswerFlag string

var reflectCmd = &cobra.Command{
	Use:   "reflect <question-number>",
	Short: "Answer a reflection question",
	Long:  "Answer one of the 6 structured reflection questions required to exit the refactor phase.",
	Example: `  tdd-ai refactor reflect 1 --answer "Tests are already descriptive and clear enough"
  tdd-ai refactor reflect 3 --answer "Each test uses its own fixture data"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := getWorkDir()
		s, err := session.LoadOrFail(dir)
		if err != nil {
			return err
		}

		if s.Phase != types.PhaseRefactor {
			return fmt.Errorf("not in refactor phase (current: %s). Reflections are only available during refactor", s.Phase)
		}

		num, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid question number %q: must be an integer", args[0])
		}

		if reflectAnswerFlag == "" {
			return fmt.Errorf("--answer is required")
		}

		if err := reflection.ValidateAnswer(reflectAnswerFlag); err != nil {
			return err
		}

		if err := s.AnswerReflection(num, reflectAnswerFlag); err != nil {
			return err
		}

		s.AddEvent("reflection_answer", func(e *types.Event) {
			e.Result = fmt.Sprintf("q%d", num)
		})

		if err := session.Save(dir, s); err != nil {
			return err
		}

		pending := s.PendingReflections()
		fmt.Fprintf(cmd.OutOrStdout(), "Answered question %d. %d remaining.\n", num, len(pending))
		return nil
	},
}

var refactorStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show all reflection questions with status",
	Long:  "Display all reflection questions with their answered/pending status.",
	Example: `  tdd-ai refactor status
  tdd-ai refactor status --format json`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		dir := getWorkDir()
		s, err := session.LoadOrFail(dir)
		if err != nil {
			return err
		}

		if s.Phase != types.PhaseRefactor {
			return fmt.Errorf("not in refactor phase (current: %s)", s.Phase)
		}

		f := formatter.Format(formatFlag)
		switch f {
		case formatter.FormatJSON:
			return renderRefactorStatusJSON(cmd, s)
		case formatter.FormatText:
			return renderRefactorStatusText(cmd, s)
		default:
			return fmt.Errorf("unknown format: %q", f)
		}
	},
}

func renderRefactorStatusJSON(cmd *cobra.Command, s *types.Session) error {
	type refactorStatusOutput struct {
		Total       int                        `json:"total"`
		Answered    int                        `json:"answered"`
		Pending     int                        `json:"pending"`
		AllAnswered bool                       `json:"all_answered"`
		Reflections []types.ReflectionQuestion `json:"reflections"`
	}

	answered := 0
	for _, r := range s.Reflections {
		if r.Answer != "" {
			answered++
		}
	}
	total := len(s.Reflections)

	out := refactorStatusOutput{
		Total:       total,
		Answered:    answered,
		Pending:     total - answered,
		AllAnswered: s.AllReflectionsAnswered(),
		Reflections: s.Reflections,
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding refactor status: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}

func renderRefactorStatusText(cmd *cobra.Command, s *types.Session) error {
	answered := 0
	for _, r := range s.Reflections {
		if r.Answer != "" {
			answered++
		}
	}
	total := len(s.Reflections)

	fmt.Fprintf(cmd.OutOrStdout(), "Reflections (%d/%d answered):\n\n", answered, total)
	for _, r := range s.Reflections {
		status := "pending"
		if r.Answer != "" {
			status = "answered"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  [%d] (%s) %s\n", r.ID, status, r.Question)
		if r.Answer != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "      -> %q\n", r.Answer)
		}
	}
	return nil
}

func init() {
	reflectCmd.Flags().StringVar(&reflectAnswerFlag, "answer", "", "your answer to the reflection question (min 5 words)")
	refactorCmd.AddCommand(reflectCmd)
	refactorCmd.AddCommand(refactorStatusCmd)
	rootCmd.AddCommand(refactorCmd)
}
