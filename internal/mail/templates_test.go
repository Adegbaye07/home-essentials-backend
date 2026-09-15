package mail_test

import (
	"strings"
	"testing"

	"homeessentials/backend/internal/mail"
)

func TestRenderDemoIncludesBrandAndFooter(t *testing.T) {
	data := mail.DemoEmailData{
		ShellData: mail.NewShellData("Test", "Hello"),
		Intro:     "Preview line.",
		ItemsTableData: mail.ItemsTableData{
			Items: []mail.LineItemRow{
				{
					ProductTitle: "Leather tote",
					Size:         "M",
					Color:        "black",
					Quantity:     2,
					UnitPrice:    mail.FormatNGN(1800000),
					LineTotal:    mail.FormatNGN(3600000),
				},
			},
		},
	}
	html, err := mail.Render("demo", data)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Home Essentials by Kamgol",
		"Leather tote",
		"black",
		"do not reply",
		"Preview line.",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered HTML missing %q", want)
		}
	}
}
