package tui

import "testing"

// TestEmptyIntensityUsesTheCLIsDefault is the bug the tmux run exposed.
//
// selectableValue defaults an empty value to the FIRST option, which for intensity
// is 1 — light. The CLI's own default is 4, so a fresh form silently proposed a
// lighter paint than the Python CLI would have produced.
func TestEmptyIntensityUsesTheCLIsDefault(t *testing.T) {
	ans := PaintTextAnswers{}
	PaintTextForm(&ans)
	if ans.Intensity != DefaultIntensityValue {
		t.Errorf("Intensity = %q, want the CLI's default %q", ans.Intensity, DefaultIntensityValue)
	}
	if ans.Intensity != "4" {
		t.Errorf("Intensity = %q, want 4", ans.Intensity)
	}
}

// TestEmptyProfileUsesTheCLIsAdvertisedDefault pins that the default comes from the
// CLI's own help text rather than from list order.
func TestEmptyProfileUsesTheCLIsAdvertisedDefault(t *testing.T) {
	t.Run("advertised default wins over list order", func(t *testing.T) {
		ans := SimulateAnswers{}
		// The default is deliberately not first, so list order cannot be mistaken
		// for the CLI's choice.
		SimulateForm(&ans, []string{"night-owl", "default", "weekend-warrior"}, "default")
		if ans.Profile != "default" {
			t.Errorf("Profile = %q, want the advertised default", ans.Profile)
		}
	})

	t.Run("falls back to the first option when nothing is advertised", func(t *testing.T) {
		ans := SimulateAnswers{}
		SimulateForm(&ans, []string{"night-owl", "default"}, "")
		if ans.Profile != "night-owl" {
			t.Errorf("Profile = %q, want the first option", ans.Profile)
		}
	})

	t.Run("an explicit choice is kept", func(t *testing.T) {
		ans := SimulateAnswers{Profile: "weekend-warrior"}
		SimulateForm(&ans, []string{"night-owl", "default", "weekend-warrior"}, "default")
		if ans.Profile != "weekend-warrior" {
			t.Errorf("Profile = %q, want the explicit choice", ans.Profile)
		}
	})
}

// TestEmptyDaysUsesTheCLIsDefault pins the documented 90-day window.
func TestEmptyDaysUsesTheCLIsDefault(t *testing.T) {
	ans := SimulateAnswers{}
	SimulateForm(&ans, []string{"default"}, "default")
	if ans.Days != "90" {
		t.Errorf("Days = %q, want 90", ans.Days)
	}
}
