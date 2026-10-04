package report

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

const site = "https://example.test/"

func sampleReport() *Report {
	r := New("0.1.2", time.Date(2026, 10, 4, 10, 0, 0, 0, time.UTC), OS{Family: "windows"})
	r.InstalledApps = []InstalledApp{{Name: `Evil</script><script>alert(1)</script>`, Scope: "user"}, {Name: "A & B", Scope: "machine"}}
	return r
}

func TestHTMLEmbedsDataSafely(t *testing.T) {
	page, err := sampleReport().HTML(site, "r.json")
	if err != nil {
		t.Fatal(err)
	}
	s := string(page)
	// Exactly one closing script tag for the data block, and none from the data.
	if strings.Count(s, "</script>") != 2 { // viewer.js tag + data block
		t.Fatalf("unexpected </script> count:\n%s", s)
	}
	m := regexp.MustCompile(`(?s)<script type="application/json" id="report-data">\n(.*)</script>`).FindStringSubmatch(s)
	if m == nil {
		t.Fatal("data block not found")
	}
	var back Report
	if err := json.Unmarshal([]byte(m[1]), &back); err != nil {
		t.Fatalf("embedded data is not valid JSON: %v", err)
	}
	if back.InstalledApps[0].Name != `Evil</script><script>alert(1)</script>` || back.InstalledApps[1].Name != "A & B" {
		t.Errorf("data did not round-trip: %+v", back.InstalledApps)
	}
}

func TestHTMLLocksThePageDown(t *testing.T) {
	page, _ := sampleReport().HTML("https://example.test", "r.json")
	s := string(page)
	for _, want := range []string{
		`connect-src &#39;none&#39;`,
		`default-src &#39;none&#39;`,
		`script-src https://example.test/viewer/;`,
		`<script src="https://example.test/viewer/viewer.js" defer></script>`,
		`<link rel="stylesheet" href="https://example.test/viewer/viewer.css">`,
		`<meta name="referrer" content="no-referrer">`,
		`https://example.test/report/`,
		`<strong>r.json</strong>`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("page missing %q", want)
		}
	}
	// No other external addresses.
	for _, u := range regexp.MustCompile(`https?://[^"'\s<;]+`).FindAllString(s, -1) {
		if !strings.HasPrefix(u, "https://example.test/") {
			t.Errorf("unexpected address %q", u)
		}
	}
}

func TestHTMLRejectsPlainHTTP(t *testing.T) {
	if _, err := sampleReport().HTML("http://example.test/", "r.json"); err == nil {
		t.Error("expected an error for a non-https site")
	}
	if _, err := sampleReport().HTML("http://localhost:4321/", "r.json"); err != nil {
		t.Errorf("localhost is allowed for testing: %v", err)
	}
}

func TestWriteHTML(t *testing.T) {
	p := filepath.Join(t.TempDir(), "ai-exposure-report-1.html")
	if err := sampleReport().WriteHTML(p, site); err != nil {
		t.Fatal(err)
	}
	if err := sampleReport().WriteHTML(p, site); err == nil {
		t.Error("should not overwrite")
	}
	if HTMLPath(`C:\x\ai-exposure-report-1.json`) != `C:\x\ai-exposure-report-1.html` || HTMLPath("/x/mine.txt") != "/x/mine.html" {
		t.Error("HTMLPath wrong")
	}
}
