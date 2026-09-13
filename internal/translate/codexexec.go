// Package translate turns a batch of English release-note bodies into
// Japanese via a single `codex exec` subprocess call, recovering the
// per-release translations from its combined output.
package translate

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// defaultCommand is the translation subprocess. Routing the request through
// codex - and from there the local LiteLLM proxy - keeps translation off the
// Claude subscription quota, whose weekly limit otherwise fails the run.
const defaultCommand = "codex"

// defaultArgs returns the fixed arguments passed to the command. The profile
// and model select the LiteLLM route; the trailing "-" makes codex read the
// prompt from stdin. A fresh slice is returned on every call so a caller
// cannot mutate the defaults for later Translators.
func defaultArgs() []string {
	return []string{
		"exec",
		"-p", "litellm",
		"-m", "muse-spark-contributor",
		"--sandbox", "read-only",
		"--skip-git-repo-check",
		"-",
	}
}

// runFunc executes command with args, writing stdin to the subprocess's
// standard input and returning its standard output (or an error, including a
// stderr excerpt, on failure). It exists so tests can substitute a fake in
// place of a real subprocess call.
type runFunc func(command string, args []string, stdin string) (stdout string, err error)

// Translator translates a batch of release-note bodies into Japanese via a
// single subprocess invocation.
type Translator struct {
	command string
	args    []string
	run     runFunc
}

// NewTranslator returns a Translator that invokes command (default "codex"
// when empty) with the fixed translation arguments, writing the assembled
// prompt to its stdin.
func NewTranslator(command string) *Translator {
	if command == "" {
		command = defaultCommand
	}
	return &Translator{command: command, args: defaultArgs(), run: runSubprocess}
}

// Translate returns one translated text per input section, in the same
// order, obtained via a single subprocess call. It returns an error - and no
// translations - when the subprocess fails, produces empty output, or any
// section cannot be recovered from the output.
func (t *Translator) Translate(sections []string) ([]string, error) {
	prompt := buildPrompt(sections)

	stdout, err := t.run(t.command, t.args, prompt)
	if err != nil {
		return nil, fmt.Errorf("translate: %w", err)
	}

	if strings.TrimSpace(stdout) == "" {
		return nil, fmt.Errorf("translate: %s produced empty output", t.command)
	}

	return splitSections(stdout, len(sections))
}

// runSubprocess is the real runFunc: it runs command with args, writing
// stdin to the subprocess's standard input and returning its standard
// output. A non-zero exit yields an error including a stderr excerpt.
func runSubprocess(command string, args []string, stdin string) (string, error) {
	cmd := exec.Command(command, args...)
	cmd.Stdin = strings.NewReader(stdin)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s failed: %w (stderr: %s)", command, err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

// promptInstruction is the fixed preamble sent ahead of every section. It
// tells the model to preserve the per-section delimiter lines verbatim so
// the combined response can be split back apart.
const promptInstruction = `You are translating Claude Code release notes into natural Japanese.

The text below contains one or more sections, each wrapped by a pair of
delimiter lines in the form:

  ◆◆◆CLAUDE-RELEASE-NOTES-SECTION-<n>-BEGIN◆◆◆
  ... section content ...
  ◆◆◆CLAUDE-RELEASE-NOTES-SECTION-<n>-END◆◆◆

Translate ONLY the content between each pair of delimiter lines into
Japanese. Reproduce every delimiter line EXACTLY as given, unchanged, in the
same order, with nothing else added before, between, or after them.

The content of each section is Markdown. Reproduce its syntax and structure
(headings, list markers, inline code, bold, links, fenced code blocks)
exactly as given, unchanged. Translate only the human-readable text; leave
the content inside inline code spans and fenced code blocks untranslated.
`

// buildPrompt assembles the full prompt: the translation instruction
// followed by every section wrapped in its own delimiter pair, in input
// order.
func buildPrompt(sections []string) string {
	var b strings.Builder
	b.WriteString(promptInstruction)
	for i, section := range sections {
		n := i + 1
		b.WriteString("\n")
		b.WriteString(beginMarker(n))
		b.WriteString("\n")
		b.WriteString(section)
		b.WriteString("\n")
		b.WriteString(endMarker(n))
		b.WriteString("\n")
	}
	return b.String()
}

// splitSections recovers count translated sections, in order, from output.
// It returns an error - and no partial results - when any section's
// delimiter pair cannot be found.
func splitSections(output string, count int) ([]string, error) {
	results := make([]string, count)
	for i := 0; i < count; i++ {
		n := i + 1
		begin := beginMarker(n)
		end := endMarker(n)

		beginIdx := strings.Index(output, begin)
		if beginIdx == -1 {
			return nil, fmt.Errorf("translate: missing begin delimiter for section %d", n)
		}
		contentStart := beginIdx + len(begin)

		endIdx := strings.Index(output[contentStart:], end)
		if endIdx == -1 {
			return nil, fmt.Errorf("translate: missing end delimiter for section %d", n)
		}

		results[i] = strings.TrimSpace(output[contentStart : contentStart+endIdx])
	}
	return results, nil
}

// beginMarker and endMarker produce the per-section delimiter lines. The
// diamond-decorated, project-specific, index-qualified form is chosen so it
// cannot plausibly occur inside real release-note text.
func beginMarker(n int) string {
	return fmt.Sprintf("◆◆◆CLAUDE-RELEASE-NOTES-SECTION-%d-BEGIN◆◆◆", n)
}

func endMarker(n int) string {
	return fmt.Sprintf("◆◆◆CLAUDE-RELEASE-NOTES-SECTION-%d-END◆◆◆", n)
}
