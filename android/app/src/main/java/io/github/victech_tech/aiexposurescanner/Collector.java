package io.github.victech_tech.aiexposurescanner;

import android.Manifest;
import android.accessibilityservice.AccessibilityServiceInfo;
import android.app.AppOpsManager;
import android.app.admin.DevicePolicyManager;
import android.content.ComponentName;
import android.content.Context;
import android.content.pm.ApplicationInfo;
import android.content.pm.PackageInfo;
import android.content.pm.PackageManager;
import android.os.Build;
import android.provider.Settings;
import android.view.accessibility.AccessibilityManager;
import android.view.inputmethod.InputMethodInfo;
import android.view.inputmethod.InputMethodManager;

import java.text.SimpleDateFormat;
import java.util.Date;
import java.util.LinkedHashMap;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Set;
import java.util.TimeZone;

/**
 * Reads the phone and builds the report. Read-only: it only asks Android
 * questions and never changes a setting, starts another app or connects to
 * the internet (the app has no internet permission at all).
 *
 * Each section is read on its own, so one failure is written to the
 * report's errors list and the rest still works.
 */
final class Collector {
    /** Ordinary permissions that Android asks the person about, and the word the report uses. */
    private static final Map<String, String> PERMISSIONS = new LinkedHashMap<>();

    static {
        PERMISSIONS.put(Manifest.permission.RECORD_AUDIO, "microphone");
        PERMISSIONS.put(Manifest.permission.CAMERA, "camera");
        PERMISSIONS.put(Manifest.permission.ACCESS_FINE_LOCATION, "location");
        PERMISSIONS.put(Manifest.permission.ACCESS_COARSE_LOCATION, "location");
        PERMISSIONS.put(Manifest.permission.READ_SMS, "sms");
        PERMISSIONS.put(Manifest.permission.RECEIVE_SMS, "sms");
        PERMISSIONS.put(Manifest.permission.READ_CALL_LOG, "call-log");
        PERMISSIONS.put(Manifest.permission.READ_CONTACTS, "contacts");
    }

    private final Context context;
    private final PackageManager pm;
    private final Report report = new Report();

    Collector(Context context) {
        this.context = context;
        this.pm = context.getPackageManager();
    }

    @SuppressWarnings("deprecation") // the int-flag methods are the ones available from Android 10
    Report collect(String version) {
        report.scannerVersion = version;
        SimpleDateFormat iso = new SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.ROOT);
        iso.setTimeZone(TimeZone.getTimeZone("UTC"));
        report.scannedAt = iso.format(new Date());
        report.osVersion = Build.VERSION.RELEASE;
        report.osEdition = maker(Build.MANUFACTURER);
        report.osArch = Build.SUPPORTED_ABIS.length > 0 ? Build.SUPPORTED_ABIS[0] : "";

        // Special access first: apps that hold it are always listed, even if
        // they came with the phone and have no icon.
        Map<String, Set<String>> special = new LinkedHashMap<>();
        accessibility(special);
        deviceAdmins(special);
        notificationAccess(special);
        keyboards(special);

        List<PackageInfo> packages;
        try {
            packages = pm.getInstalledPackages(PackageManager.GET_PERMISSIONS);
        } catch (RuntimeException e) {
            report.errors.add(new Report.Problem("installedApps", "Android did not give the list of apps."));
            return report;
        }

        boolean appOpsFailed = false;
        AppOpsManager appOps = context.getSystemService(AppOpsManager.class);
        for (PackageInfo pi : packages) {
            ApplicationInfo ai = pi.applicationInfo;
            if (ai == null || pi.packageName.equals(context.getPackageName())) continue;
            boolean preinstalled = (ai.flags & (ApplicationInfo.FLAG_SYSTEM | ApplicationInfo.FLAG_UPDATED_SYSTEM_APP)) != 0;
            Set<String> access = special.get(pi.packageName);
            // Phones carry hundreds of hidden system parts. Keep the ones a
            // person would recognise (they have an icon) or that hold special access.
            if (preinstalled && access == null && pm.getLaunchIntentForPackage(pi.packageName) == null) continue;

            String label = label(ai);
            report.installedApps.add(new Report.App(label, pi.packageName, pi.versionName, preinstalled, installer(pi.packageName)));

            if (access != null) {
                for (String capability : access) report.privacyPermissions.add(new Report.Permission(capability, label, pi.packageName));
            }
            Set<String> granted = new LinkedHashSet<>();
            boolean wantsOverlay = false, wantsUsage = false, wantsBoot = false;
            if (pi.requestedPermissions != null) {
                for (int i = 0; i < pi.requestedPermissions.length; i++) {
                    String p = pi.requestedPermissions[i];
                    boolean isGranted = pi.requestedPermissionsFlags != null
                            && (pi.requestedPermissionsFlags[i] & PackageInfo.REQUESTED_PERMISSION_GRANTED) != 0;
                    if (isGranted && PERMISSIONS.containsKey(p)) granted.add(PERMISSIONS.get(p));
                    if (Manifest.permission.SYSTEM_ALERT_WINDOW.equals(p)) wantsOverlay = true;
                    if (Manifest.permission.PACKAGE_USAGE_STATS.equals(p)) wantsUsage = true;
                    if (isGranted && Manifest.permission.RECEIVE_BOOT_COMPLETED.equals(p)) wantsBoot = true;
                }
            }
            if (!preinstalled && appOps != null && !appOpsFailed) {
                try {
                    if (wantsOverlay && allowed(appOps, AppOpsManager.OPSTR_SYSTEM_ALERT_WINDOW, ai.uid, pi.packageName)) {
                        granted.add("display-over-apps");
                    }
                    if (wantsUsage && allowed(appOps, AppOpsManager.OPSTR_GET_USAGE_STATS, ai.uid, pi.packageName)) {
                        granted.add("usage-access");
                    }
                } catch (SecurityException e) {
                    appOpsFailed = true;
                    report.errors.add(new Report.Problem("privacyPermissions",
                            "Android did not say which apps can draw over other apps or see app usage."));
                }
            }
            for (String capability : granted) report.privacyPermissions.add(new Report.Permission(capability, label, pi.packageName));
            if (wantsBoot && !preinstalled) report.startupItems.add(new Report.Startup(label, pi.packageName));
        }
        return report;
    }

    @SuppressWarnings("deprecation") // the replacement only exists on the newest Android versions
    private static boolean allowed(AppOpsManager appOps, String op, int uid, String pkg) {
        return appOps.unsafeCheckOpNoThrow(op, uid, pkg) == AppOpsManager.MODE_ALLOWED;
    }

    /** Apps with accessibility access can see the screen and tap for you. */
    private void accessibility(Map<String, Set<String>> special) {
        try {
            AccessibilityManager am = context.getSystemService(AccessibilityManager.class);
            for (AccessibilityServiceInfo info : am.getEnabledAccessibilityServiceList(AccessibilityServiceInfo.FEEDBACK_ALL_MASK)) {
                if (info.getResolveInfo() != null && info.getResolveInfo().serviceInfo != null) {
                    add(special, info.getResolveInfo().serviceInfo.packageName, "accessibility");
                }
            }
        } catch (RuntimeException e) {
            report.errors.add(new Report.Problem("accessibility", "Android did not say which apps have accessibility access."));
        }
    }

    /** Device admin apps can lock or wipe the phone and can be hard to remove. */
    private void deviceAdmins(Map<String, Set<String>> special) {
        try {
            DevicePolicyManager dpm = context.getSystemService(DevicePolicyManager.class);
            List<ComponentName> admins = dpm.getActiveAdmins();
            if (admins != null) for (ComponentName c : admins) add(special, c.getPackageName(), "device-admin");
        } catch (RuntimeException e) {
            report.errors.add(new Report.Problem("deviceAdmins", "Android did not say which apps are device admin apps."));
        }
    }

    /** Apps with notification access can read every notification, including message previews. */
    private void notificationAccess(Map<String, Set<String>> special) {
        try {
            String list = Settings.Secure.getString(context.getContentResolver(), "enabled_notification_listeners");
            if (list == null) return;
            for (String flat : list.split(":")) {
                ComponentName c = ComponentName.unflattenFromString(flat);
                if (c != null) add(special, c.getPackageName(), "notification-access");
            }
        } catch (RuntimeException e) {
            report.errors.add(new Report.Problem("notificationAccess", "Android did not say which apps can read notifications."));
        }
    }

    /** Keyboards see everything typed with them. */
    private void keyboards(Map<String, Set<String>> special) {
        try {
            InputMethodManager imm = context.getSystemService(InputMethodManager.class);
            for (InputMethodInfo info : imm.getEnabledInputMethodList()) add(special, info.getPackageName(), "keyboard");
        } catch (RuntimeException e) {
            report.errors.add(new Report.Problem("keyboards", "Android did not give the list of keyboards."));
        }
    }

    private static void add(Map<String, Set<String>> special, String pkg, String capability) {
        if (pkg == null) return;
        Set<String> set = special.get(pkg);
        if (set == null) {
            set = new LinkedHashSet<>();
            special.put(pkg, set);
        }
        set.add(capability);
    }

    private String label(ApplicationInfo ai) {
        try {
            CharSequence l = pm.getApplicationLabel(ai);
            if (l != null && l.length() > 0) return l.toString();
        } catch (RuntimeException ignored) {
            // fall back to the package name below
        }
        return ai.packageName;
    }

    @SuppressWarnings("deprecation") // getInstallerPackageName is the only way on Android 10
    private String installer(String pkg) {
        try {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
                return pm.getInstallSourceInfo(pkg).getInstallingPackageName();
            }
            return pm.getInstallerPackageName(pkg);
        } catch (PackageManager.NameNotFoundException | RuntimeException e) {
            return null;
        }
    }

    /** "samsung" becomes "Samsung". */
    static String maker(String s) {
        if (s == null || s.isEmpty()) return "";
        return s.substring(0, 1).toUpperCase(Locale.ROOT) + s.substring(1);
    }
}
