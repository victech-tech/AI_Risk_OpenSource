# Privacy

The AI Exposure Scanner **makes no network connections and collects no data.**

- It does not connect to the internet, and has no update checks, usage tracking (telemetry) or crash reporting.
- It reads only the locations listed in the README under "What the scanner reads".
- It saves one report file on your own computer. You decide whether to use it.
- The report never contains passwords, API keys, tokens, file contents, browsing history or your Windows user name.
- At the end it asks Windows to open the report page in your browser. Only the web address is passed; no report data is included. Use `--no-browser` to skip this.

When you drop the report onto the website, it is read **inside your browser** and is not uploaded. The report page loads no third-party scripts and its security settings block it from sending data anywhere.

This is enforced by tests in `internal/policy/` that fail if network code is added to the scanner.
