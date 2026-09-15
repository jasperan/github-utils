package tui

import (
	"strings"
	"testing"

	"github.com/jasperan/github-utils/gotui/internal/huhstyle"
)

// The accessibility contract, and its one honest limitation.
//
// huh v2.0.3 consults WithAccessible only inside Form.RunWithContext (form.go:676);
// Form.Init/Update/View never read the flag. This front-end EMBEDS its forms as
// Bubble Tea components, so the full-screen UI cannot serve a screen reader, and
// WithAccessible on an embedded form is inert.
//
// The correct response is the one taken here: route ACCESSIBLE to the plain
// renderer instead of pretending the TUI is accessible. These tests pin that.

// TestAccessibleIsDetectedFromTheEnvironment pins the single switch that selects
// the screen-reader path.
func TestAccessibleIsDetectedFromTheEnvironment(t *testing.T) {
	t.Setenv("ACCESSIBLE", "")
	if huhstyle.Accessible() {
		t.Error("Accessible() = true with ACCESSIBLE unset")
	}
	t.Setenv("ACCESSIBLE", "1")
	if !huhstyle.Accessible() {
		t.Error("Accessible() = false with ACCESSIBLE=1")
	}
	// Any non-empty value counts, because the variable is a flag rather than a
	// boolean to parse.
	t.Setenv("ACCESSIBLE", "yes")
	if !huhstyle.Accessible() {
		t.Error("Accessible() = false with ACCESSIBLE=yes")
	}
}

// TestAccessiblePathProducesNoStyling is the screen-reader guarantee: the plain
// output must be free of escape sequences and box drawing that a reader would
// announce as noise.
func TestAccessiblePathProducesNoStyling(t *testing.T) {
	t.Setenv("ACCESSIBLE", "1")
	runner := &stubRunner{preview: []byte(previewFixture), plan: []byte(planFixture)}
	opts := testOptions(runner)

	report, err := RunPlain(opts, PlainAction{Kind: PlainPaintText, Text: "HI", Intensity: 4})
	if err != nil {
		t.Fatalf("RunPlain: %v", err)
	}
	if strings.ContainsRune(report, 0x1b) {
		t.Error("the accessible path emitted an ANSI escape")
	}
	for _, noise := range []string{"╭", "╰", "┏", "┃", "▐"} {
		if strings.Contains(report, noise) {
			t.Errorf("the accessible path emitted decorative box drawing %q", noise)
		}
	}
	// The numbers a reader needs must still be present.
	if !strings.Contains(report, "total estimated commits") {
		t.Errorf("the accessible path dropped the totals:\n%s", report)
	}
}

// TestAccessibleUsageIsSelfContained checks that the refusal a screen-reader user
// gets explains the way forward without relying on the interactive UI.
func TestAccessibleUsageIsSelfContained(t *testing.T) {
	t.Setenv("ACCESSIBLE", "1")
	opts := testOptions(&stubRunner{})
	usage := PlainUsage(opts)

	if !strings.Contains(usage, "No terminal is available") {
		t.Errorf("usage should explain why nothing is prompted for:\n%s", usage)
	}
	for _, flag := range []string{"--paint", "--pattern", "--simulate", "--before", "--after"} {
		if !strings.Contains(usage, flag) {
			t.Errorf("usage is missing %s, leaving no route for a non-interactive user", flag)
		}
	}
}

// TestFormsCarryTheAccessibleFlagForIntent documents that the flag is set even
// though an embedded form ignores it.
//
// This is deliberately not an assertion about behaviour: huh ignores the flag here,
// so claiming otherwise would be a lie in a test as well as in the UI. What is
// pinned is that the embedded form still builds and renders normally with the flag
// set, i.e. setting it cannot break the TUI.
func TestFormsCarryTheAccessibleFlagForIntent(t *testing.T) {
	t.Setenv("ACCESSIBLE", "1")

	ans := PaintTextAnswers{Text: "HI"}
	form := PaintTextForm(&ans)
	if form == nil {
		t.Fatal("no form was built with ACCESSIBLE set")
	}
	// Rendering must not fail or panic; the embedded path is still the one in use.
	if view := form.View(); view == "" {
		t.Error("the embedded form rendered nothing with ACCESSIBLE set")
	}
}

// TestModelRendersWithAccessibleSet keeps the flag from changing TUI behaviour in
// a way that would break the interactive path for everyone else.
func TestModelRendersWithAccessibleSet(t *testing.T) {
	t.Setenv("ACCESSIBLE", "1")

	runner := &stubRunner{preview: []byte(previewFixture), plan: []byte(planFixture)}
	m := New(testOptions(runner))
	m.Update(m.paintTextCmd()())

	if view := m.View().Content; view == "" {
		t.Error("the model rendered nothing with ACCESSIBLE set")
	}
}
