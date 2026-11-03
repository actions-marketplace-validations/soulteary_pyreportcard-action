package report

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/soulteary/pyreportcard-action/internal/model"
)

func TestBadgeStyleAndGrades(t *testing.T) {
	for grade, color := range map[string]string{"A+": "44cc11", "A": "97ca00", "B": "a4a61d", "C": "dfb317", "D": "fe7d37", "F": "e05d44"} {
		svg := SVG(model.Result{Complete: true, Mode: "standard", Grade: grade})
		var badge struct {
			Width  int `xml:"width,attr"`
			Height int `xml:"height,attr"`
		}
		if err := xml.Unmarshal([]byte(svg), &badge); err != nil {
			t.Fatal(err)
		}
		if badge.Height != 20 || badge.Width >= 180 || !strings.Contains(svg, `rx="3"`) || !strings.Contains(svg, `fill="#`+color+`"`) {
			t.Errorf("unexpected badge for %s: %s", grade, svg)
		}
	}
}
func TestBadgeAnalysisStateAndEscaping(t *testing.T) {
	standard := SVG(model.Result{Complete: true, Mode: "standard", Grade: "A+"})
	typed := SVG(model.Result{Complete: true, Mode: "typed", Grade: "A+"})
	if standard == typed || !strings.Contains(typed, "python report (typed)") {
		t.Fatal("typed mode must remain visibly distinct")
	}
	incomplete := SVG(model.Result{Mode: "standard", Grade: "A+"})
	if strings.Contains(incomplete, ">A+<") || !strings.Contains(incomplete, "incomplete") || !strings.Contains(incomplete, "#9f9f9f") {
		t.Fatal("incomplete analysis must not show a grade")
	}
	var parsed any
	if err := xml.Unmarshal([]byte(SVG(model.Result{Complete: true, Mode: `<&"`, Grade: `<&"`})), &parsed); err != nil {
		t.Fatal(err)
	}
}
