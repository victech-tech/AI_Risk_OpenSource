# AI Exposure Scanner for Android

The Android version of the scanner. Like the Windows scanner, it **only collects facts** (which apps are on the phone and what special access Android has given them), saves a report file and lets the website explain it **inside the browser**. It never judges risk itself: the rules in `../rules/` do that on the website.

- **One permission:** `QUERY_ALL_PACKAGES`, so it can see the list of apps.
- **No internet permission at all.** Android itself stops the app connecting to anything. `internal/policy/android_test.go` and the `Android app` CI workflow fail if that ever changes.
- **No libraries.** Plain Java and the Android framework. JUnit and AndroidX Test are used for tests only and are not in the app.
- **Read-only.** It never changes a setting, removes an app or starts another app (except opening the report page in your browser when you tap the button).
- Android 10 (API 29) and later. Targets Android 16 (API 36), as Google Play requires.

The complete list of what it reads is in the main [README](../README.md#android-app).

## How it works for the user

1. Open the app. It explains what it reads and what it never does. Nothing happens until **Start the check** is tapped.
2. It takes a few seconds, then saves `ai-exposure-report-YYYYMMDD-HHMM.json` in **Downloads**.
3. **Open the results page** opens the website's `/report/` page in the browser. The person taps **Choose file** and picks the report. The browser reads it on the phone; nothing is uploaded.
4. **Send the report to myself** shares the file (for example by email to their own computer). **See exactly what the report contains** shows the file in the app.

## Code

```
app/src/main/java/.../MainActivity.java   the one screen: explanation, start, results buttons
app/src/main/java/.../Collector.java      reads the phone (each section on its own; failures go in errors)
app/src/main/java/.../Report.java         the report (schema/report.schema.json, version 1), plain Java
app/src/main/java/.../Json.java           a tiny JSON writer, so no library is needed
app/src/test/                             unit tests (run on any computer)
app/src/androidTest/                      on-device test of the real collector (CI runs it in an emulator)
```

`ReportTest` checks that the app writes exactly `../testdata/reports/example-android.json`, which CI also validates against the report schema, so the app and the schema cannot drift apart. To regenerate the example after a deliberate change: `UPDATE_EXAMPLE=1 ./gradlew testDebugUnitTest`.

## Build and test

Needs JDK 17 or later and the Android SDK (platform 36, build tools 36). Point Gradle at the SDK with `ANDROID_HOME` or a `local.properties` file containing `sdk.dir=/path/to/sdk`.

```
./gradlew testDebugUnitTest        # unit tests
./gradlew lintRelease              # Android lint (must be clean)
./gradlew assembleDebug            # app-debug.apk, installable for testing
./gradlew connectedDebugAndroidTest   # on a connected phone or emulator
```

Install a test build on your own phone with `adb install app/build/outputs/apk/debug/app-debug.apk`, or copy the file across and open it (Android asks you to allow installing from that source).

`REPORT_URL` (an `https://` address) changes where **Open the results page** goes; otherwise it uses the default in `app/build.gradle.kts`.

## Releases and signing

Pushing an `android-v*` tag (for example `android-v0.1.0`) runs `.github/workflows/android-release.yml`. It tests, builds the `.apk` (for direct install) and `.aab` (for Google Play), checks the permissions, writes `checksums-android.txt` and build attestations. The release is **never marked "latest"**, because the website's Windows download links use `releases/latest`.

Android only installs signed apps. Make an upload key once, keep it safe (losing it means you cannot update the app outside Google Play):

```
keytool -genkeypair -v -keystore upload.jks -keyalg RSA -keysize 4096 -validity 10000 -alias upload
base64 -w0 upload.jks     # paste the output into the ANDROID_KEYSTORE_BASE64 secret
```

Repository secrets: `ANDROID_KEYSTORE_BASE64`, `ANDROID_KEYSTORE_PASSWORD`, `ANDROID_KEY_ALIAS` (for example `upload`), `ANDROID_KEY_PASSWORD`. Without them the workflow keeps unsigned files as workflow artifacts only and makes no GitHub release.

## Google Play

- A Google Play developer account costs a one-off US$25. New personal accounts must run a closed test with at least 12 testers for 14 days before going public.
- Use **Play App Signing**: upload the `.aab`; Google signs what users download, and your key above becomes the upload key.
- `QUERY_ALL_PACKAGES` needs a declaration in Play Console (App content > Sensitive permissions). The fitting use is "device search / security": the app's core purpose is to show people which apps on their phone can see or control it. Say clearly that the app has no internet permission and nothing leaves the phone.
- Data safety form: the app collects and shares **no** data (it has no internet permission). The report is saved on the phone by the user's action.
- The store listing must not claim the phone is "safe" or "clean"; use "shows known AI tools" wording, as on the website.
