package translate

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// wellFormedResponse builds a fake claude-batch stdout response containing
// one delimited section per entry in translations, padded with whitespace
// around each marker/content to exercise trimming.
func wellFormedResponse(translations []string) string {
	var b strings.Builder
	for i, tr := range translations {
		n := i + 1
		b.WriteString("  ")
		b.WriteString(beginMarker(n))
		b.WriteString("\n  ")
		b.WriteString(tr)
		b.WriteString("  \n")
		b.WriteString(endMarker(n))
		b.WriteString("  \n\n")
	}
	return b.String()
}

func TestNewTranslator_DefaultCommand(t *testing.T) {
	tr := NewTranslator("")
	if tr.command != defaultCommand {
		t.Fatalf("expected default command %q, got %q", defaultCommand, tr.command)
	}
}

func TestNewTranslator_CustomCommand(t *testing.T) {
	tr := NewTranslator("my-fake-claude-batch")
	if tr.command != "my-fake-claude-batch" {
		t.Fatalf("expected command %q, got %q", "my-fake-claude-batch", tr.command)
	}
}

// TestTranslate_PromptAssembly references AC-1: the assembled prompt
// contains the instruction, every input section, and one delimiter pair per
// section, in input order.
func TestTranslate_PromptAssembly(t *testing.T) {
	sections := []string{"Section one body text.", "Section two body text."}

	var capturedStdin string
	tr := NewTranslator("fake-command")
	tr.run = func(command, arg, stdin string) (string, error) {
		capturedStdin = stdin
		return wellFormedResponse([]string{"translated one", "translated two"}), nil
	}

	if _, err := tr.Translate(sections); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(capturedStdin, promptInstruction) {
		t.Fatalf("prompt does not contain the translation instruction: %q", capturedStdin)
	}

	idx0 := strings.Index(capturedStdin, sections[0])
	idx1 := strings.Index(capturedStdin, sections[1])
	if idx0 == -1 {
		t.Fatalf("prompt missing section 0 content")
	}
	if idx1 == -1 {
		t.Fatalf("prompt missing section 1 content")
	}
	if idx0 > idx1 {
		t.Fatalf("sections out of order in prompt: section 0 at %d, section 1 at %d", idx0, idx1)
	}

	for i := range sections {
		n := i + 1
		if !strings.Contains(capturedStdin, beginMarker(n)) {
			t.Fatalf("prompt missing begin marker for section %d", n)
		}
		if !strings.Contains(capturedStdin, endMarker(n)) {
			t.Fatalf("prompt missing end marker for section %d", n)
		}
	}
}

// TestTranslate_OutputRecovery references AC-2: a well-formed fake response
// is split into the same number of translations, in order, with delimiters
// and padding whitespace removed.
func TestTranslate_OutputRecovery(t *testing.T) {
	sections := []string{"Body A", "Body B", "Body C"}
	want := []string{"訳A", "訳B", "訳C"}

	tr := NewTranslator("fake-command")
	tr.run = func(command, arg, stdin string) (string, error) {
		return wellFormedResponse(want), nil
	}

	got, err := tr.Translate(sections)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d translations, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("translation %d: expected %q, got %q", i, want[i], got[i])
		}
	}
}

// TestTranslate_SubprocessError references AC-3: subprocess non-zero exit
// yields an error; no translations are returned.
func TestTranslate_SubprocessError(t *testing.T) {
	tr := NewTranslator("fake-command")
	tr.run = func(command, arg, stdin string) (string, error) {
		return "", fmt.Errorf("claude-batch failed: %w (stderr: boom)", errors.New("exit status 1"))
	}

	got, err := tr.Translate([]string{"Body"})
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
	if got != nil {
		t.Fatalf("expected no translations on error, got %v", got)
	}
}

// TestTranslate_EmptyOutput references AC-4: empty stdout yields an error.
func TestTranslate_EmptyOutput(t *testing.T) {
	tr := NewTranslator("fake-command")
	tr.run = func(command, arg, stdin string) (string, error) {
		return "   \n\t", nil
	}

	got, err := tr.Translate([]string{"Body"})
	if err == nil {
		t.Fatalf("expected an error for empty output, got nil")
	}
	if got != nil {
		t.Fatalf("expected no translations on error, got %v", got)
	}
}

// TestTranslate_MissingDelimiter references AC-5: a response missing one
// section's delimiters yields an error (never a silently shorter result).
func TestTranslate_MissingDelimiter(t *testing.T) {
	sections := []string{"Body A", "Body B"}

	tr := NewTranslator("fake-command")
	tr.run = func(command, arg, stdin string) (string, error) {
		// Only section 1's delimiters are present; section 2 is missing.
		return wellFormedResponse([]string{"訳A"}), nil
	}

	got, err := tr.Translate(sections)
	if err == nil {
		t.Fatalf("expected an error for a missing delimiter, got nil (result: %v)", got)
	}
	if got != nil {
		t.Fatalf("expected no translations on error, got %v", got)
	}
}

// TestTranslate_InvocationArgs references AC-6: the subprocess is invoked
// with exactly the argument "-" and the prompt on stdin.
func TestTranslate_InvocationArgs(t *testing.T) {
	sections := []string{"Body A"}

	var gotCommand, gotArg, gotStdin string
	tr := NewTranslator("fake-command-name")
	tr.run = func(command, arg, stdin string) (string, error) {
		gotCommand, gotArg, gotStdin = command, arg, stdin
		return wellFormedResponse([]string{"訳A"}), nil
	}

	if _, err := tr.Translate(sections); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotCommand != "fake-command-name" {
		t.Fatalf("expected command %q, got %q", "fake-command-name", gotCommand)
	}
	if gotArg != "-" {
		t.Fatalf("expected arg %q, got %q", "-", gotArg)
	}
	if !strings.Contains(gotStdin, sections[0]) {
		t.Fatalf("expected stdin to contain the section body, got %q", gotStdin)
	}
}
