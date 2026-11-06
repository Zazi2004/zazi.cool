package habit

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestWeekdays(t *testing.T) {
	w := Weekdays(0b0010101) // Monday, Wednesday, Friday
	want := map[time.Weekday]bool{
		time.Monday: true, time.Tuesday: false, time.Wednesday: true, time.Thursday: false,
		time.Friday: true, time.Saturday: false, time.Sunday: false,
	}
	for d, has := range want {
		if w.Has(d) != has {
			t.Errorf("Has(%v) = %v, want %v", d, w.Has(d), has)
		}
	}
	if !Weekdays(0b1000000).Has(time.Sunday) {
		t.Error("bit 6 should be Sunday")
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		in, want string
		ok       bool
	}{
		{"Lire", "Lire", true},
		{"  Lire  ", "Lire", true},
		{"<script>", "<script>", true},
		{"", "", false},
		{"   ", "", false},
		{strings.Repeat("é", 60), strings.Repeat("é", 60), true},
		{strings.Repeat("é", 61), "", false},
		{"a\nb", "", false},
	}
	for _, tt := range tests {
		got, err := ValidateName(tt.in)
		if (err == nil) != tt.ok || got != tt.want {
			t.Errorf("ValidateName(%q) = %q, %v; want %q, ok=%v", tt.in, got, err, tt.want, tt.ok)
		}
	}
}

func validInput() Input {
	return Input{
		Name: "Vitamine D", Emoji: "💊", Color: "amber", Moment: Morning,
		Unit: "pilules", Step: 1, Period: PeriodDay, Target: 2, Weekdays: AllDays,
	}
}

func TestInputValidate(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Input)
		ok     bool
	}{
		{"valid", func(*Input) {}, true},
		{"yes/no habit", func(in *Input) { in.Unit, in.Target = "", 1 }, true},
		{"empty name", func(in *Input) { in.Name = "" }, false},
		{"unknown color", func(in *Input) { in.Color = "mauve" }, false},
		{"unknown moment", func(in *Input) { in.Moment = "night" }, false},
		{"unknown period", func(in *Input) { in.Period = "decade" }, false},
		{"empty emoji", func(in *Input) { in.Emoji = "" }, false},
		{"letters as emoji", func(in *Input) { in.Emoji = "ab" }, false},
		{"family emoji", func(in *Input) { in.Emoji = "👨‍👩‍👧‍👦" }, true},
		{"target 0", func(in *Input) { in.Target = 0 }, false},
		{"target 10000", func(in *Input) { in.Target = 10000 }, true},
		{"target 10001", func(in *Input) { in.Target = 10001 }, false},
		{"step 0", func(in *Input) { in.Step = 0 }, false},
		{"unit of 20", func(in *Input) { in.Unit = strings.Repeat("é", 20) }, true},
		{"unit of 21", func(in *Input) { in.Unit = strings.Repeat("é", 21) }, false},
		{"no weekday", func(in *Input) { in.Weekdays = 0 }, false},
		{"bit 7 set", func(in *Input) { in.Weekdays = 0b10000000 }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validInput()
			tt.change(&in)
			_, err := in.Validate()
			if (err == nil) != tt.ok {
				t.Fatalf("Validate() error = %v, want ok=%v", err, tt.ok)
			}
			var ve *ValidationError
			if err != nil && !errors.As(err, &ve) {
				t.Errorf("error %T is not a *ValidationError", err)
			}
		})
	}
}

func TestInputValidateTrims(t *testing.T) {
	in := validInput()
	in.Name, in.Unit = "  Vitamine D ", " pilules "
	got, err := in.Validate()
	if err != nil || got.Name != "Vitamine D" || got.Unit != "pilules" {
		t.Errorf("Validate() = %+v, %v", got, err)
	}
}

func TestParseDay(t *testing.T) {
	if _, err := ParseDay("2026-10-30"); err != nil {
		t.Errorf("valid day: %v", err)
	}
	for _, s := range []string{"", "2026-1-5", "2026-02-30", "30/10/2026", "x"} {
		if _, err := ParseDay(s); err == nil {
			t.Errorf("ParseDay(%q): want an error", s)
		}
	}
}

// 23:30 UTC on 30 June is 1 July in Paris.
func TestTodayUsesTheConfiguredZone(t *testing.T) {
	paris, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.June, 30, 23, 30, 0, 0, time.UTC)
	if got := FormatDay(Today(now, paris)); got != "2026-07-01" {
		t.Errorf("Today = %s, want 2026-07-01", got)
	}
	if got := FormatDay(Today(now, time.UTC)); got != "2026-06-30" {
		t.Errorf("Today in UTC = %s, want 2026-06-30", got)
	}
}
