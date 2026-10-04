package io.github.victech_tech.aiexposurescanner;

import java.util.ArrayList;
import java.util.List;

/**
 * The report, in the same format as the Windows scanner
 * (schema/report.schema.json, version 1). Plain Java with no Android code,
 * so it can be unit tested on any computer.
 *
 * It holds facts only: app names, package names and which special access
 * Android gives each app. Never messages, files, accounts or anything typed.
 */
public final class Report {
    public static final int MAX_TEXT = 200;

    public String scannerVersion = "";
    /** ISO 8601 time in UTC, for example 2026-10-04T12:00:00Z. */
    public String scannedAt = "";
    public String osVersion = "";
    /** The phone maker, for example Samsung. Never the model or serial number. */
    public String osEdition = "";
    public String osArch = "";

    public final List<App> installedApps = new ArrayList<>();
    public final List<Startup> startupItems = new ArrayList<>();
    public final List<Permission> privacyPermissions = new ArrayList<>();
    public final List<Problem> errors = new ArrayList<>();

    public static final class App {
        public final String name;
        public final String id;
        public final String version;
        /** "user" for apps the person installed, "machine" for apps that came with the phone. */
        public final String scope;
        /** Package name of the store that installed it, or null. */
        public final String installer;

        public App(String name, String id, String version, boolean preinstalled, String installer) {
            this.name = name;
            this.id = id;
            this.version = version;
            this.scope = preinstalled ? "machine" : "user";
            this.installer = installer;
        }
    }

    public static final class Startup {
        public final String name;
        public final String id;

        public Startup(String name, String id) {
            this.name = name;
            this.id = id;
        }
    }

    public static final class Permission {
        public final String capability;
        public final String app;
        public final String appId;

        public Permission(String capability, String app, String appId) {
            this.capability = capability;
            this.app = app;
            this.appId = appId;
        }
    }

    public static final class Problem {
        public final String section;
        public final String message;

        public Problem(String section, String message) {
            this.section = section;
            this.message = message;
        }
    }

    /** Writes the report as indented JSON, matching the Windows scanner's layout. */
    public String toJson() {
        Json j = new Json();
        j.beginObject();
        j.name("schemaVersion").value(1);
        j.name("scanner").beginObject();
        j.name("name").value("ai-exposure-scanner");
        j.name("version").value(scannerVersion);
        j.endObject();
        j.name("scannedAt").value(scannedAt);
        j.name("os").beginObject();
        j.name("family").value("android");
        optional(j, "version", osVersion);
        optional(j, "edition", osEdition);
        optional(j, "arch", osArch);
        j.endObject();

        j.name("installedApps").beginArray();
        for (App a : installedApps) {
            j.beginObject();
            j.name("name").value(clip(a.name));
            optional(j, "version", clip(a.version));
            j.name("scope").value(a.scope);
            j.name("id").value(clip(a.id));
            optional(j, "installer", clip(a.installer));
            j.endObject();
        }
        j.endArray();

        j.name("startupItems").beginArray();
        for (Startup s : startupItems) {
            j.beginObject();
            j.name("name").value(clip(s.name));
            j.name("source").value("android-boot");
            j.name("command").value(clip(s.id));
            j.endObject();
        }
        j.endArray();

        j.name("privacyPermissions").beginArray();
        for (Permission p : privacyPermissions) {
            j.beginObject();
            j.name("capability").value(p.capability);
            j.name("app").value(clip(p.app));
            j.name("appId").value(clip(p.appId));
            j.name("allowed").value(true);
            j.endObject();
        }
        j.endArray();

        if (!errors.isEmpty()) {
            j.name("errors").beginArray();
            for (Problem e : errors) {
                j.beginObject();
                j.name("section").value(e.section);
                j.name("message").value(e.message);
                j.endObject();
            }
            j.endArray();
        }
        j.endObject();
        return j.toString();
    }

    private static void optional(Json j, String name, String value) {
        if (value != null && !value.isEmpty()) j.name(name).value(value);
    }

    /** Trims text to the schema's 200-character limit without splitting a character. */
    static String clip(String s) {
        if (s == null) return null;
        s = s.trim();
        if (s.codePointCount(0, s.length()) <= MAX_TEXT) return s;
        return s.substring(0, s.offsetByCodePoints(0, MAX_TEXT));
    }
}
