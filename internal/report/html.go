package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
)

// HTML builds a small results page that shows the report in the browser
// with one double-click. The report data is embedded in the page; the page
// loads its display code (viewer.js and viewer.css) from siteURL. The
// page's own Content Security Policy lets it load only those two files and
// blocks every connection, so the data cannot leave the computer.
//
// siteURL is the website's address with a trailing slash, for example
// https://ai-exposure-check.vinvictech.workers.dev/
func (r *Report) HTML(siteURL, jsonName string) ([]byte, error) {
	if !strings.HasSuffix(siteURL, "/") {
		siteURL += "/"
	}
	if !strings.HasPrefix(siteURL, "https://") && !strings.HasPrefix(siteURL, "http://localhost") {
		return nil, fmt.Errorf("site address must use https: %q", siteURL)
	}
	// json.Marshal escapes <, > and & as \u003c etc., so the data can never
	// close the <script> element it sits in.
	var data bytes.Buffer
	enc := json.NewEncoder(&data)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		return nil, err
	}
	viewer := siteURL + "viewer/"
	reportPage := siteURL + "report/"
	csp := fmt.Sprintf("default-src 'none'; script-src %s; style-src %s; img-src data:; connect-src 'none'; base-uri 'none'; form-action 'none'", viewer, viewer)
	e := html.EscapeString

	var b strings.Builder
	b.WriteString("<!doctype html>\n<html lang=\"en-GB\">\n<head>\n<meta charset=\"utf-8\">\n")
	fmt.Fprintf(&b, "<meta http-equiv=\"Content-Security-Policy\" content=\"%s\">\n", e(csp))
	b.WriteString("<meta name=\"referrer\" content=\"no-referrer\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	b.WriteString("<title>Your AI exposure results</title>\n")
	fmt.Fprintf(&b, "<link rel=\"stylesheet\" href=\"%sviewer.css\">\n", e(viewer))
	fmt.Fprintf(&b, "<script src=\"%sviewer.js\" defer></script>\n", e(viewer))
	b.WriteString("</head>\n<body>\n<main id=\"fallback\">\n<h1>Your AI exposure results</h1>\n")
	b.WriteString("<p>Loading your results. This page needs an internet connection to show them. Your data stays in this file on your computer and is not uploaded.</p>\n")
	fmt.Fprintf(&b, "<p>If nothing appears, go to <a href=\"%s\">%s</a> and drag the file <strong>%s</strong> (in the same folder as this page) onto it.</p>\n",
		e(reportPage), e(reportPage), e(jsonName))
	b.WriteString("</main>\n<script type=\"application/json\" id=\"report-data\">\n")
	b.Write(data.Bytes())
	b.WriteString("</script>\n</body>\n</html>\n")
	return []byte(b.String()), nil
}

// WriteHTML saves the results page next to the report. It refuses to
// overwrite an existing file.
func (r *Report) WriteHTML(path, siteURL string) error {
	jsonName := strings.TrimSuffix(filepath.Base(path), ".html") + ".json"
	data, err := r.HTML(siteURL, jsonName)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// HTMLPath gives the results page path for a report path:
// report.json -> report.html.
func HTMLPath(jsonPath string) string {
	return strings.TrimSuffix(jsonPath, filepath.Ext(jsonPath)) + ".html"
}
