package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/macosta/tdd-ai/internal/claims"
	"github.com/macosta/tdd-ai/internal/formatter"
	"github.com/macosta/tdd-ai/internal/observe"
	"github.com/spf13/cobra"
)

// Seams so tests can stand in for the tracer and the clock.
var (
	traceAvailable = observe.Available
	runTraced      = observe.Run
	runPlain       = observe.RunPlain
	claimClock     = time.Now
)

var (
	claimPinFlags        []string
	claimNoteFlag        string
	claimAllowReadFlags  []string
	claimAllowWriteFlags []string
	claimAllowExtFlags   []string
	claimAllowNetFlags   []string
	claimAllowExecFlags  []string
	claimNoStdoutFlag    bool
	claimStrictFlag      bool
	claimNoTraceFlag     bool
	claimTimeoutFlag     time.Duration
)

const minNoteWords = 5

var claimCmd = &cobra.Command{
	Use:   "claim",
	Short: "Verify a claim once, then check it on every run",
	Long: `Claims turn a plain-language statement about a command into checks that run
without a model.

  1. claim add      record the claim and the command that exercises it
  2. claim verify   run the command under a tracer and record what it touched
  3. claim distill  after judging the evidence, turn it into deterministic checks
  4. claim check    rerun the checks in CI and in every agent loop

tdd-ai records what it observed, not what an agent reports. Distill refuses
evidence that contradicts the claim unless you accept it explicitly: network
activity, files outside the project and other programs all need a flag.

Tracing uses strace, so verify needs Linux. Check can run with --no-trace
anywhere to compare pinned files, the exit code and the output.`,
	Example: `  tdd-ai claim add "reconcile only reads ./invoices and never touches the network" \
    --pin skills/invoice/SKILL.md --pin skills/invoice/reconcile.mjs \
    -- node skills/invoice/reconcile.mjs invoices/march.csv
  tdd-ai claim verify 1
  tdd-ai claim distill 1 --note "no network activity and reads stay in invoices and the skill"
  tdd-ai claim check`,
}

var claimAddCmd = &cobra.Command{
	Use:   `add "claim" [--pin file]... -- <command> [args...]`,
	Short: "Record a claim and the command that exercises it",
	Long: `Record a plain-language claim and, after --, the command that exercises it.

Pin the files the claim depends on, such as a skill's SKILL.md and scripts. If a
pinned file changes, the claim's checks fail until it is verified again.`,
	Example: `  tdd-ai claim add "reconcile never touches the network" \
    --pin skills/invoice/reconcile.mjs -- node skills/invoice/reconcile.mjs invoices/march.csv`,
	Args: func(cmd *cobra.Command, args []string) error {
		if cmd.ArgsLenAtDash() != 1 || len(args) < 2 {
			return errors.New(`give the claim, then -- and the command that exercises it: tdd-ai claim add "claim" -- <command> [args...]`)
		}
		if strings.TrimSpace(args[0]) == "" {
			return errors.New("the claim text is empty")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		dir := getWorkDir()
		pins := []string{}
		for _, p := range claimPinFlags {
			clean, err := claims.CleanPin(p)
			if err != nil {
				return err
			}
			if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(clean))); err != nil {
				return fmt.Errorf("pinned file %s not found", clean)
			}
			if !containsString(pins, clean) {
				pins = append(pins, clean)
			}
		}

		f, err := claims.Load(dir)
		if err != nil {
			return err
		}
		c := f.Add(strings.TrimSpace(args[0]), args[1:], pins, claimClock())
		if err := claims.Save(dir, f); err != nil {
			return err
		}

		return writeClaimOutput(cmd, c, func(w io.Writer) {
			fmt.Fprintf(w, "Claim [%d] added: %s\n", c.ID, c.Text)
			fmt.Fprintf(w, "  run:  %s\n", strings.Join(c.Run, " "))
			fmt.Fprintf(w, "  pins: %s\n", joinOrNone(c.Pins))
			fmt.Fprintf(w, "Next: run 'tdd-ai claim verify %d' to observe it\n", c.ID)
		})
	},
}

var claimListCmd = &cobra.Command{
	Use:   "list",
	Short: "List claims and their status",
	Example: `  tdd-ai claim list
  tdd-ai claim list --format json`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		f, err := claims.Load(getWorkDir())
		if err != nil {
			return err
		}
		return writeClaimOutput(cmd, map[string]any{"claims": f.Claims}, func(w io.Writer) {
			if len(f.Claims) == 0 {
				fmt.Fprintln(w, `No claims yet. Add one with 'tdd-ai claim add "claim" -- <command>'`)
				return
			}
			for _, c := range f.Claims {
				fmt.Fprintf(w, "[%d] %-10s %s\n", c.ID, c.Status, c.Text)
				fmt.Fprintf(w, "     run: %s\n", strings.Join(c.Run, " "))
			}
		})
	},
}

var claimVerifyCmd = &cobra.Command{
	Use:   "verify <id>",
	Short: "Run a claim's command under a tracer and record the evidence",
	Long: `Runs the claim's command under strace and records what it actually did: files
read and written inside and outside the project, network activity, programs it
ran, its exit code and its output. Nothing here is self-reported.

Then judge the claim against the evidence and the pinned files. If it holds,
run 'tdd-ai claim distill <id> --note "why"'.`,
	Example: `  tdd-ai claim verify 1
  tdd-ai claim verify 1 --format json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		id, err := parseClaimID(args[0])
		if err != nil {
			return err
		}
		dir := getWorkDir()
		f, c, err := loadClaim(dir, id)
		if err != nil {
			return err
		}
		if !traceAvailable() {
			return errors.New("claim verify observes the command with strace, which needs Linux. On macOS or Windows, run it in a Linux container or in CI")
		}
		pins := claims.HashPins(dir, c.Pins)
		for _, p := range c.Pins {
			if pins[p] == "" {
				return fmt.Errorf("pinned file %s is missing", p)
			}
		}

		ctx, cancel := context.WithTimeout(commandContext(cmd), claimTimeoutFlag)
		defer cancel()
		res, err := runTraced(ctx, dir, c.Run)
		if err != nil {
			return err
		}

		now := claimClock()
		ev := claims.FromResult(claims.DefaultLocations(dir), res, pins, "strace", now)
		if err := writeProjectFile(dir, claims.ObservedOutputPath(id), res.Stdout); err != nil {
			return err
		}
		hadChecks := c.Checks != nil
		c.Evidence = ev
		c.Status = claims.StatusObserved
		if err := claims.Save(dir, f); err != nil {
			return err
		}

		proposed, needs := claims.Distill(ev, claims.Options{Run: c.Run, StdoutPath: claims.ExpectedOutputPath(id), Now: now})
		notable := claims.Notable(ev)
		next := fmt.Sprintf(`judge the claim against this evidence and the pinned files. If it holds, run 'tdd-ai claim distill %d --note "why it holds"'`, id)

		out := map[string]any{
			"claim":           c,
			"notable":         orEmptyStrings(notable),
			"proposed_checks": proposed,
			"needs_decision":  orEmptyStrings(needs),
			"next":            next,
		}
		return writeClaimOutput(cmd, out, func(w io.Writer) {
			fmt.Fprintf(w, "Observed claim [%d]: %s\n", c.ID, c.Text)
			fmt.Fprintf(w, "  ran:      %s (exit %d)\n", strings.Join(c.Run, " "), ev.ExitCode)
			fmt.Fprintf(w, "  read:     %s\n", joinOrNone(ev.Reads))
			fmt.Fprintf(w, "  wrote:    %s\n", joinOrNone(ev.Writes))
			fmt.Fprintf(w, "  outside:  %s\n", joinOrNone(append(append([]string{}, ev.ExternalReads...), ev.ExternalWrites...)))
			fmt.Fprintf(w, "  network:  %s\n", joinOrNone(ev.Network))
			fmt.Fprintf(w, "  programs: %s\n", joinOrNone(ev.Execs))
			fmt.Fprintf(w, "  output:   %d bytes, saved to %s\n", ev.StdoutBytes, claims.ObservedOutputPath(id))
			if len(notable) == 0 {
				fmt.Fprintln(w, "\nNothing notable: no network activity, nothing outside the project, no writes, no other programs.")
			} else {
				fmt.Fprintln(w, "\nNotable:")
				for _, n := range notable {
					fmt.Fprintf(w, "  - %s\n", n)
				}
			}
			fmt.Fprintln(w, "\nProposed checks:")
			writeChecks(w, proposed)
			if len(needs) > 0 {
				fmt.Fprintln(w, "\nNeeds a decision before distilling:")
				for _, n := range needs {
					fmt.Fprintf(w, "  - %s\n", n)
				}
			}
			if hadChecks {
				fmt.Fprintln(w, "\nThe claim's earlier checks stay in force until you distill again.")
			}
			fmt.Fprintf(w, "\nNext: %s\n", next)
		})
	},
}

var claimDistillCmd = &cobra.Command{
	Use:   "distill <id> --note \"why the evidence backs the claim\"",
	Short: "Turn a claim's accepted evidence into deterministic checks",
	Long: `Records your verdict and turns the claim's evidence into checks: pinned file
hashes, the read and write policy, the allowed network and programs, the exit
code and the expected output (saved under .tdd-ai/expected/).

Distill refuses evidence that needs a decision: network activity, files outside
the project and other programs must be accepted with --allow-net,
--allow-external and --allow-exec. --allow-read and --allow-write replace the
proposed globs and must cover everything that was observed.`,
	Example: `  tdd-ai claim distill 1 --note "no network activity and reads stay in invoices and the skill"
  tdd-ai claim distill 2 --note "talks only to the local database as documented" --allow-net 127.0.0.1:5432`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		id, err := parseClaimID(args[0])
		if err != nil {
			return err
		}
		dir := getWorkDir()
		f, c, err := loadClaim(dir, id)
		if err != nil {
			return err
		}
		if c.Evidence == nil {
			return fmt.Errorf("claim %d has not been observed yet. Run 'tdd-ai claim verify %d' first", id, id)
		}
		note := strings.TrimSpace(claimNoteFlag)
		if len(strings.Fields(note)) < minNoteWords {
			return fmt.Errorf("--note must say why the evidence backs the claim (at least %d words)", minNoteWords)
		}

		checks, refusals := claims.Distill(c.Evidence, claims.Options{
			Reads:      claimAllowReadFlags,
			Writes:     claimAllowWriteFlags,
			External:   claimAllowExtFlags,
			Network:    claimAllowNetFlags,
			Execs:      claimAllowExecFlags,
			SkipStdout: claimNoStdoutFlag,
			Run:        c.Run,
			StdoutPath: claims.ExpectedOutputPath(id),
			PinsNow:    claims.HashPins(dir, c.Pins),
			Now:        claimClock(),
		})
		if len(refusals) > 0 {
			if err := writeClaimOutput(cmd, map[string]any{"distilled": false, "claim_id": id, "refusals": refusals}, func(w io.Writer) {
				fmt.Fprintf(w, "Claim [%d] was not distilled:\n", id)
				for _, r := range refusals {
					fmt.Fprintf(w, "  - %s\n", r)
				}
			}); err != nil {
				return err
			}
			return fmt.Errorf("distill refused: %d issue(s) need a decision", len(refusals))
		}

		if checks.Stdout != "" {
			observed, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(claims.ObservedOutputPath(id))))
			if err != nil || claims.HashBytes(observed) != c.Evidence.StdoutSHA256 {
				return fmt.Errorf("the observed output for claim %d is missing or changed. Run 'tdd-ai claim verify %d' again", id, id)
			}
			if err := writeProjectFile(dir, checks.Stdout, observed); err != nil {
				return err
			}
		}

		c.Checks = checks
		c.Note = note
		c.Status = claims.StatusVerified
		if err := claims.Save(dir, f); err != nil {
			return err
		}

		return writeClaimOutput(cmd, map[string]any{"distilled": true, "claim": c}, func(w io.Writer) {
			fmt.Fprintf(w, "Claim [%d] verified: %s\n", c.ID, c.Text)
			writeChecks(w, checks)
			fmt.Fprintln(w, "Next: run 'tdd-ai claim check' in CI and in every agent loop")
		})
	},
}

type claimCheckResult struct {
	ID      int    `json:"id"`
	Text    string `json:"text"`
	Status  string `json:"status"`
	Pending bool   `json:"pending"`
	claims.Outcome
}

var claimCheckCmd = &cobra.Command{
	Use:   "check [id...]",
	Short: "Rerun the distilled checks; exit 1 if any fails",
	Long: `Reruns each verified claim's command and compares it with the distilled checks:
pinned file hashes, exit code, expected output and, under the tracer, the read,
write, network and program policy. No model is involved, so it is cheap enough
for CI and every agent loop.

Claims that were never distilled are reported as pending; --strict fails on
them. --no-trace skips the tracer (for macOS and Windows) and checks pins, the
exit code and the output only.`,
	Example: `  tdd-ai claim check
  tdd-ai claim check 1 2 --format json
  tdd-ai claim check --no-trace`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		dir := getWorkDir()
		f, err := claims.Load(dir)
		if err != nil {
			return err
		}
		selected, err := selectClaims(f, args)
		if err != nil {
			return err
		}
		traced := !claimNoTraceFlag
		if traced && !traceAvailable() {
			return errors.New("claim check traces each command with strace, which needs Linux. Pass --no-trace to check pinned files, the exit code and the output only")
		}

		loc := claims.DefaultLocations(dir)
		results := []claimCheckResult{}
		passed, failed, pending := 0, 0, 0
		for _, c := range selected {
			r := claimCheckResult{ID: c.ID, Text: c.Text, Status: string(c.Status)}
			if c.Checks == nil {
				r.Pending = true
				r.Failures = []claims.Failure{}
				pending++
				results = append(results, r)
				continue
			}
			r.Outcome = checkClaim(cmd, dir, loc, c, traced)
			if r.Pass {
				passed++
			} else {
				failed++
			}
			results = append(results, r)
		}

		pass := failed == 0 && (!claimStrictFlag || pending == 0)
		out := map[string]any{
			"pass": pass, "traced": traced,
			"passed": passed, "failed": failed, "pending": pending,
			"results": results,
		}
		if err := writeClaimOutput(cmd, out, func(w io.Writer) {
			if len(results) == 0 {
				fmt.Fprintln(w, `No claims yet. Add one with 'tdd-ai claim add "claim" -- <command>'`)
				return
			}
			for _, r := range results {
				switch {
				case r.Pending:
					fmt.Fprintf(w, "PENDING [%d] %s (not distilled yet)\n", r.ID, r.Text)
				case r.Pass:
					fmt.Fprintf(w, "PASS    [%d] %s\n", r.ID, r.Text)
				default:
					fmt.Fprintf(w, "FAIL    [%d] %s\n", r.ID, r.Text)
					for _, fl := range r.Failures {
						fmt.Fprintf(w, "          %s: %s\n", fl.Check, fl.Detail)
					}
				}
			}
			if !traced {
				fmt.Fprintln(w, "Policy checks skipped (--no-trace): reads, writes, network and programs were not compared.")
			}
			fmt.Fprintf(w, "%d passed, %d failed, %d pending\n", passed, failed, pending)
		}); err != nil {
			return err
		}
		if !pass {
			if failed == 0 {
				return fmt.Errorf("%d claim(s) pending and --strict is set", pending)
			}
			return fmt.Errorf("%d claim(s) failed", failed)
		}
		return nil
	},
}

func checkClaim(cmd *cobra.Command, dir string, loc claims.Locations, c *claims.Claim, traced bool) claims.Outcome {
	ctx, cancel := context.WithTimeout(commandContext(cmd), claimTimeoutFlag)
	defer cancel()

	run, observer := runTraced, "strace"
	if !traced {
		run, observer = runPlain, "none"
	}
	res, err := run(ctx, dir, c.Run)
	if err != nil {
		return claims.Outcome{Failures: []claims.Failure{{Check: "run", Detail: err.Error()}}, PolicyChecked: traced}
	}

	var expected []byte
	if c.Checks.Stdout != "" {
		if data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(c.Checks.Stdout))); err == nil {
			expected = data
		}
	}
	pinned := make([]string, 0, len(c.Checks.Pins))
	for p := range c.Checks.Pins {
		pinned = append(pinned, p)
	}
	sort.Strings(pinned)

	return claims.Evaluate(c.Checks, claims.Observation{
		Run:      c.Run,
		Evidence: claims.FromResult(loc, res, nil, observer, claimClock()),
		Traced:   traced,
		PinsNow:  claims.HashPins(dir, pinned),
		Stdout:   res.Stdout,
		Expected: expected,
	})
}

func selectClaims(f *claims.File, args []string) ([]*claims.Claim, error) {
	if len(args) == 0 {
		out := make([]*claims.Claim, 0, len(f.Claims))
		for i := range f.Claims {
			out = append(out, &f.Claims[i])
		}
		return out, nil
	}
	out := make([]*claims.Claim, 0, len(args))
	for _, a := range args {
		id, err := parseClaimID(a)
		if err != nil {
			return nil, err
		}
		c, err := f.Get(id)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func parseClaimID(s string) (int, error) {
	id, err := strconv.Atoi(s)
	if err != nil || id < 1 {
		return 0, fmt.Errorf("claim ID must be a positive number, got %q", s)
	}
	return id, nil
}

func loadClaim(dir string, id int) (*claims.File, *claims.Claim, error) {
	f, err := claims.Load(dir)
	if err != nil {
		return nil, nil, err
	}
	c, err := f.Get(id)
	if err != nil {
		return nil, nil, err
	}
	return f, c, nil
}

func commandContext(cmd *cobra.Command) context.Context {
	if ctx := cmd.Context(); ctx != nil {
		return ctx
	}
	return context.Background()
}

func writeProjectFile(dir, rel string, data []byte) error {
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(rel), err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", rel, err)
	}
	return nil
}

// writeClaimOutput prints v as JSON or calls text, depending on --format.
func writeClaimOutput(cmd *cobra.Command, v any, text func(io.Writer)) error {
	switch formatter.Format(formatFlag) {
	case formatter.FormatJSON:
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return fmt.Errorf("encoding output: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
	case formatter.FormatText:
		text(cmd.OutOrStdout())
	default:
		return fmt.Errorf("unknown format: %q", formatFlag)
	}
	return nil
}

func writeChecks(w io.Writer, ch *claims.Checks) {
	fmt.Fprintf(w, "  command:  %s\n", strings.Join(ch.Run, " "))
	pins := make([]string, 0, len(ch.Pins))
	for p := range ch.Pins {
		pins = append(pins, p)
	}
	sort.Strings(pins)
	fmt.Fprintf(w, "  pins:     %s\n", joinOrNone(pins))
	fmt.Fprintf(w, "  reads:    %s\n", joinOrNone(ch.Reads))
	fmt.Fprintf(w, "  writes:   %s\n", joinOrNone(ch.Writes))
	fmt.Fprintf(w, "  outside:  %s\n", joinOrNone(ch.External))
	fmt.Fprintf(w, "  network:  %s\n", joinOrNone(ch.Network))
	fmt.Fprintf(w, "  programs: %s\n", joinOrNone(ch.Execs))
	fmt.Fprintf(w, "  exit:     %d\n", ch.ExitCode)
	if ch.Stdout != "" {
		fmt.Fprintf(w, "  output:   must match %s\n", ch.Stdout)
	} else {
		fmt.Fprintln(w, "  output:   not checked")
	}
}

func joinOrNone(list []string) string {
	if len(list) == 0 {
		return "none"
	}
	return strings.Join(list, ", ")
}

func orEmptyStrings(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func init() {
	claimAddCmd.Flags().StringArrayVar(&claimPinFlags, "pin", nil, "file the claim depends on; its hash is checked on every run (repeatable)")

	claimVerifyCmd.Flags().DurationVar(&claimTimeoutFlag, "timeout", 2*time.Minute, "stop the command after this long")
	claimCheckCmd.Flags().DurationVar(&claimTimeoutFlag, "timeout", 2*time.Minute, "stop each command after this long")

	claimDistillCmd.Flags().StringVar(&claimNoteFlag, "note", "", "why the evidence backs the claim (required, at least 5 words)")
	claimDistillCmd.Flags().StringArrayVar(&claimAllowReadFlags, "allow-read", nil, "project glob allowed to be read; replaces the proposed reads (repeatable)")
	claimDistillCmd.Flags().StringArrayVar(&claimAllowWriteFlags, "allow-write", nil, "project glob allowed to be written; replaces the proposed writes (repeatable)")
	claimDistillCmd.Flags().StringArrayVar(&claimAllowExtFlags, "allow-external", nil, "path outside the project allowed to be read or written, e.g. '~/.config/app/**' (repeatable)")
	claimDistillCmd.Flags().StringArrayVar(&claimAllowNetFlags, "allow-net", nil, "network entry allowed, e.g. 127.0.0.1:5432 (repeatable)")
	claimDistillCmd.Flags().StringArrayVar(&claimAllowExecFlags, "allow-exec", nil, "other program the command may run, by name (repeatable)")
	claimDistillCmd.Flags().BoolVar(&claimNoStdoutFlag, "no-stdout", false, "do not check the command's output")

	claimCheckCmd.Flags().BoolVar(&claimStrictFlag, "strict", false, "fail when any selected claim has not been distilled")
	claimCheckCmd.Flags().BoolVar(&claimNoTraceFlag, "no-trace", false, "skip the tracer: check pinned files, exit code and output only")

	claimCmd.AddCommand(claimAddCmd, claimListCmd, claimVerifyCmd, claimDistillCmd, claimCheckCmd)
	rootCmd.AddCommand(claimCmd)
}
