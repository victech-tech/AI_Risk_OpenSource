package io.github.victech_tech.aiexposurescanner;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertTrue;

import java.io.File;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;

import org.junit.Test;

public class ReportTest {
    /** The example report in testdata/reports/, which CI also checks against schema/report.schema.json. */
    static final File EXAMPLE = new File("../../testdata/reports/example-android.json");

    static Report example() {
        Report r = new Report();
        r.scannerVersion = "0.1.0";
        r.scannedAt = "2026-10-04T12:00:00Z";
        r.osVersion = "14";
        r.osEdition = "Samsung";
        r.osArch = "arm64-v8a";
        r.installedApps.add(new Report.App("ChatGPT", "com.openai.chatgpt", "1.2025.266", false, "com.android.vending"));
        r.installedApps.add(new Report.App("AnyDesk", "com.anydesk.anydeskandroid", "7.0.1", false, "com.android.vending"));
        r.installedApps.add(new Report.App("System Helper", "com.example.helper", "2.1", false, null));
        r.installedApps.add(new Report.App("Gemini", "com.google.android.apps.bard", "1.0", true, null));
        r.installedApps.add(new Report.App("Samsung Keyboard", "com.samsung.android.honeyboard", "5.6", true, null));
        r.startupItems.add(new Report.Startup("AnyDesk", "com.anydesk.anydeskandroid"));
        r.startupItems.add(new Report.Startup("System Helper", "com.example.helper"));
        r.privacyPermissions.add(new Report.Permission("microphone", "ChatGPT", "com.openai.chatgpt"));
        r.privacyPermissions.add(new Report.Permission("accessibility", "AnyDesk", "com.anydesk.anydeskandroid"));
        r.privacyPermissions.add(new Report.Permission("accessibility", "System Helper", "com.example.helper"));
        r.privacyPermissions.add(new Report.Permission("device-admin", "System Helper", "com.example.helper"));
        r.privacyPermissions.add(new Report.Permission("notification-access", "System Helper", "com.example.helper"));
        r.privacyPermissions.add(new Report.Permission("location", "System Helper", "com.example.helper"));
        r.privacyPermissions.add(new Report.Permission("keyboard", "Samsung Keyboard", "com.samsung.android.honeyboard"));
        r.errors.add(new Report.Problem("privacyPermissions", "Android did not say which apps can draw over other apps or see app usage."));
        return r;
    }

    @Test
    public void matchesTheExampleReport() throws Exception {
        String json = example().toJson();
        if (System.getenv("UPDATE_EXAMPLE") != null) Files.write(EXAMPLE.toPath(), json.getBytes(StandardCharsets.UTF_8));
        String expected = new String(Files.readAllBytes(EXAMPLE.toPath()), StandardCharsets.UTF_8);
        assertEquals(expected, json);
    }

    @Test
    public void leavesOutEmptyOptionalFields() {
        Report r = new Report();
        r.scannerVersion = "0.1.0";
        r.installedApps.add(new Report.App("App", "com.example.app", null, false, null));
        String json = r.toJson();
        assertFalse(json.contains("installer"));
        assertFalse(json.contains("\"version\": \"\""));
        assertFalse(json.contains("errors"));
        assertTrue(json.contains("\"privacyPermissions\": []"));
    }

    @Test
    public void escapesText() {
        Report r = new Report();
        r.installedApps.add(new Report.App("Quote \" back \\ new\nline \u0001 end", "com.example.app", "1", false, null));
        assertTrue(r.toJson().contains("\"Quote \\\" back \\\\ new\\nline \\u0001 end\""));
    }

    @Test
    public void clipsLongNamesWithoutSplittingCharacters() {
        StringBuilder s = new StringBuilder();
        for (int i = 0; i < 250; i++) s.append("😀");
        String clipped = Report.clip(s.toString());
        assertEquals(Report.MAX_TEXT, clipped.codePointCount(0, clipped.length()));
        assertEquals("abc", Report.clip("  abc  "));
    }

    @Test
    public void namesTheMaker() {
        assertEquals("Samsung", Collector.maker("samsung"));
        assertEquals("", Collector.maker(""));
    }
}
