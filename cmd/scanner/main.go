// Command ai-exposure-scanner lists the AI tools, remote access tools and
// related settings on a Windows PC and saves them to a report file. It is
// read-only and makes no network connections at all. The user drops the
// report onto the website, which analyses it inside their browser.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/victech-tech/AI_Risk_OpenSource/internal/collect"
	"github.com/victech-tech/AI_Risk_OpenSource/internal/report"
)

// Set at build time by GoReleaser (see .goreleaser.yaml).
var (
	version   = "dev"
	reportURL = "https://ai-exposure-check.vinvictech.workers.dev/report/"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout))
}

type options struct {
	yes       bool
	out       string
	noBrowser bool
	version   bool
}

func parseFlags(args []string, stderr io.Writer) (options, error) {
	var o options
	fs := flag.NewFlagSet("ai-exposure-scanner", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.BoolVar(&o.yes, "yes", false, "start without asking first, and close without waiting")
	fs.StringVar(&o.out, "out", "", "where to save the report (a file or a folder)")
	fs.BoolVar(&o.noBrowser, "no-browser", false, "do not open the report page or File Explorer at the end")
	fs.BoolVar(&o.version, "version", false, "show the version and exit")
	err := fs.Parse(args)
	return o, err
}

func run(args []string, stdin io.Reader, stdout io.Writer) int {
	opts, err := parseFlags(args, stdout)
	if err != nil {
		return 2
	}
	if opts.version {
		fmt.Fprintf(stdout, "ai-exposure-scanner %s\n", version)
		return 0
	}
	in := bufio.NewReader(stdin)
	now := time.Now()
	outPath, err := resolveOutPath(opts.out, now)
	if err != nil {
		fmt.Fprintf(stdout, "Could not work out where to save the report: %v\n", err)
		return 1
	}

	printIntro(stdout, outPath)
	if !opts.yes {
		fmt.Fprint(stdout, "Press Y then Enter to start, or just Enter to cancel: ")
		answer, _ := in.ReadString('\n')
		if !strings.EqualFold(strings.TrimSpace(answer), "y") {
			fmt.Fprintln(stdout, "\nCancelled. Nothing was read.")
			waitForEnter(stdout, in, opts)
			return 0
		}
	}

	r := scan(stdout, now)

	if err := r.Write(outPath); err != nil {
		fmt.Fprintf(stdout, "\nCould not save the report: %v\n", err)
		waitForEnter(stdout, in, opts)
		return 1
	}
	pagePath := report.HTMLPath(outPath)
	if pagePath == outPath {
		pagePath = outPath + ".html"
	}
	if err := r.WriteHTML(pagePath, siteURL()); err != nil {
		// The .json report is still saved; the website can read it.
		fmt.Fprintf(stdout, "\n(Could not save the results page: %v)\n", err)
		pagePath = ""
	}

	printFinish(stdout, outPath, pagePath, !opts.noBrowser)
	if !opts.noBrowser {
		if pagePath != "" {
			openInBrowser(pagePath)
		} else {
			openInBrowser(reportURL)
		}
	}
	waitForEnter(stdout, in, opts)
	return 0
}

func printIntro(w io.Writer, outPath string) {
	fmt.Fprintf(w, `AI Exposure Check - scanner %s

This program makes a list of the AI tools and related settings on this PC,
so you can see which ones can see or control your computer.

What it reads:
  - the names of installed apps and programs running now
  - programs that start automatically when Windows starts
  - which apps Windows allows to use your microphone, camera, location
    and screen capture
  - browser and code editor extensions (names and permissions only)
  - the settings of known AI tools (only the names of their tool connections)

What it will never do:
  - change, delete or install anything
  - upload anything or connect to the internet
  - read your documents, emails, passwords, browsing history or cookies
  - copy API keys, tokens or other secrets

The report will be saved here:
  %s

`, version, outPath)
}

// siteURL is the website's address, taken from reportURL
// (https://example/report/ -> https://example/).
func siteURL() string {
	return strings.TrimSuffix(strings.TrimSuffix(reportURL, "/"), "/report") + "/"
}

// printFinish tells people where their results are. The scanner itself
// never judges risk (the website's code does, inside the results page), so
// this screen must make the next step impossible to miss.
func printFinish(w io.Writer, jsonPath, pagePath string, browser bool) {
	line := strings.Repeat("=", 64)
	if pagePath != "" && browser {
		fmt.Fprintf(w, "\n%s\n Scan finished. Your results are opening in your browser.\n%s\n\n", line, line)
	} else {
		fmt.Fprintf(w, "\n%s\n Scan finished. Your report is ready.\n%s\n\n", line, line)
	}
	if pagePath != "" {
		fmt.Fprintln(w, "Your results page:")
		fmt.Fprintf(w, "  %s\n", pagePath)
		fmt.Fprintln(w, "  (double-click it to open it again at any time)")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "It shows you:")
		fmt.Fprintln(w, "  - which AI tools can see or control this PC")
		fmt.Fprintln(w, "  - how to switch each one off")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Your data stays on this PC. The page loads only its display code")
		fmt.Fprintln(w, "from our website and is blocked from sending anything anywhere.")
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, "The raw data is saved here (you can open it in Notepad):")
	fmt.Fprintf(w, "  %s\n", jsonPath)
	if pagePath == "" {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "To see your results, go to %s and drag that file onto the page.\n", reportURL)
	} else {
		fmt.Fprintf(w, "If the results page does not work, go to %s\nand drag the .json file onto it.\n", reportURL)
	}
}

func scan(w io.Writer, now time.Time) *report.Report {
	env := collect.EnvFromOS()
	r := report.New(version, now, collect.OSInfo())
	fmt.Fprintln(w)
	for _, s := range collect.Sections() {
		fmt.Fprintf(w, "Checking: %s ... ", s.Label)
		before := len(r.Errors)
		collect.RunSection(s, env, r)
		if len(r.Errors) > before && sectionEmpty(r, s.Name) {
			fmt.Fprintln(w, "skipped (see the errors list in the report)")
		} else {
			fmt.Fprintln(w, "done")
		}
	}
	return r
}

func sectionEmpty(r *report.Report, name string) bool {
	switch name {
	case "installedApps":
		return len(r.InstalledApps) == 0
	case "runningProcesses":
		return len(r.RunningProcesses) == 0
	case "startupItems":
		return len(r.StartupItems) == 0
	case "privacyPermissions":
		return len(r.PrivacyPermissions) == 0
	case "browserExtensions":
		return len(r.BrowserExtensions) == 0
	case "aiToolConfigs":
		return len(r.AIToolConfigs) == 0
	}
	return true
}

// resolveOutPath works out the report file path. With no --out it uses the
// Downloads folder. If --out is a folder, the default file name is used.
func resolveOutPath(out string, now time.Time) (string, error) {
	name := report.FileName(now)
	if out == "" {
		dir, err := downloadsDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(dir, name), nil
	}
	if fi, err := os.Stat(out); err == nil && fi.IsDir() {
		return filepath.Join(out, name), nil
	}
	if strings.HasSuffix(out, `\`) || strings.HasSuffix(out, "/") {
		return filepath.Join(out, name), nil
	}
	return filepath.Abs(out)
}

func waitForEnter(w io.Writer, in *bufio.Reader, opts options) {
	if opts.yes {
		return
	}
	fmt.Fprint(w, "\nPress Enter to close.")
	_, _ = in.ReadString('\n')
}
