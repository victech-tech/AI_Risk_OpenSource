# Privacy

The AI Exposure Scanner **makes no network connections and collects no data.**

- It does not connect to the internet, and has no update checks, usage tracking (telemetry) or crash reporting.
- It reads only the locations listed in the README under "What the scanner reads".
- It saves two files on your own computer: the report (`.json`) and a results page (`.html`) that shows it.
- When you open the results page, your browser loads only its display code from the AI Exposure Check website. The report stays inside the file. The page's security settings (a Content Security Policy) allow nothing else to load and block every connection, so the data cannot be sent anywhere.
- The report never contains passwords, API keys, tokens, file contents, browsing history or your Windows user name.
- At the end it asks Windows to open the results page in your browser. Use `--no-browser` to skip this.

## Android app

The Android app is held to the same promise, and Android enforces it: the app has **no internet permission**, so it cannot connect to anything. Its only permission lets it see the list of apps on the phone. It saves the report in your Downloads folder. If you tap "Open the results page", the app asks your browser to open the website's report page, and you choose the file yourself. The app reads only what the README lists under "Android app", and never messages, notification contents, photos, files, accounts or anything you type.

## The website

When you drop the report onto the website, it is read **inside your browser** and is not uploaded. The report page loads no third-party scripts and its security settings block it from sending data anywhere.

This is enforced by tests in `internal/policy/` that fail if network code is added to the scanner, or if the Android app asks for any permission other than listing apps. CI also checks the permissions of the built Android app.
