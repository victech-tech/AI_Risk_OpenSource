package io.github.victech_tech.aiexposurescanner;

import android.app.Activity;
import android.content.ActivityNotFoundException;
import android.content.ContentResolver;
import android.content.ContentValues;
import android.content.Intent;
import android.net.Uri;
import android.os.Bundle;
import android.os.Environment;
import android.provider.MediaStore;
import android.view.View;
import android.widget.Button;
import android.widget.TextView;
import android.widget.Toast;

import java.io.IOException;
import java.io.OutputStream;
import java.nio.charset.StandardCharsets;
import java.text.SimpleDateFormat;
import java.util.Date;
import java.util.Locale;

/**
 * One screen in three parts: what the check reads (and never does), a
 * "working" message, then where the report was saved and what to do next.
 * Nothing starts until the person taps "Start the check".
 */
public class MainActivity extends Activity {
    private View intro, working, done;
    private TextView doneSummary, reportText;
    private Uri savedUri;
    private String savedName;

    @Override
    protected void onCreate(Bundle state) {
        super.onCreate(state);
        setContentView(R.layout.activity_main);
        intro = findViewById(R.id.intro);
        working = findViewById(R.id.working);
        done = findViewById(R.id.done);
        doneSummary = findViewById(R.id.done_summary);
        reportText = findViewById(R.id.report_text);

        ((TextView) findViewById(R.id.version)).setText(getString(R.string.version, BuildConfig.VERSION_NAME));
        findViewById(R.id.start).setOnClickListener(v -> start());
        findViewById(R.id.open_results).setOnClickListener(v -> openResultsPage());
        findViewById(R.id.share).setOnClickListener(v -> share());
        Button show = findViewById(R.id.show_report);
        show.setOnClickListener(v -> {
            boolean showing = reportText.getVisibility() == View.VISIBLE;
            reportText.setVisibility(showing ? View.GONE : View.VISIBLE);
            show.setText(showing ? R.string.show_report : R.string.hide_report);
        });
    }

    private void start() {
        intro.setVisibility(View.GONE);
        working.setVisibility(View.VISIBLE);
        new Thread(() -> {
            Report report = new Collector(getApplicationContext()).collect(BuildConfig.VERSION_NAME);
            String json = report.toJson();
            String name = "ai-exposure-report-" + new SimpleDateFormat("yyyyMMdd-HHmm", Locale.ROOT).format(new Date()) + ".json";
            Uri uri = save(name, json);
            runOnUiThread(() -> finished(report, json, name, uri));
        }).start();
    }

    /** Saves the report in Downloads. Needs no permission from Android 10. */
    private Uri save(String name, String json) {
        ContentResolver resolver = getContentResolver();
        ContentValues values = new ContentValues();
        values.put(MediaStore.Downloads.DISPLAY_NAME, name);
        values.put(MediaStore.Downloads.MIME_TYPE, "application/json");
        values.put(MediaStore.Downloads.RELATIVE_PATH, Environment.DIRECTORY_DOWNLOADS);
        Uri uri = null;
        try {
            uri = resolver.insert(MediaStore.Downloads.EXTERNAL_CONTENT_URI, values);
            if (uri == null) return null;
            try (OutputStream out = resolver.openOutputStream(uri)) {
                if (out == null) throw new IOException("no stream");
                out.write(json.getBytes(StandardCharsets.UTF_8));
            }
            return uri;
        } catch (IOException | RuntimeException e) {
            if (uri != null) resolver.delete(uri, null, null);
            return null;
        }
    }

    private void finished(Report report, String json, String name, Uri uri) {
        working.setVisibility(View.GONE);
        done.setVisibility(View.VISIBLE);
        savedUri = uri;
        savedName = name;
        int apps = report.installedApps.size();
        String summary = getResources().getQuantityString(R.plurals.apps_found, apps, apps);
        if (uri != null) {
            summary += "\n\n" + getString(R.string.saved_as, name);
        } else {
            summary += "\n\n" + getString(R.string.not_saved);
            findViewById(R.id.open_results).setEnabled(false);
            findViewById(R.id.share).setEnabled(false);
        }
        doneSummary.setText(summary);
        reportText.setText(json);
    }

    /** Opens the website's report page in the browser. The app itself sends nothing. */
    private void openResultsPage() {
        Toast.makeText(this, getString(R.string.pick_file_hint, savedName), Toast.LENGTH_LONG).show();
        try {
            startActivity(new Intent(Intent.ACTION_VIEW, Uri.parse(BuildConfig.REPORT_URL)));
        } catch (ActivityNotFoundException e) {
            Toast.makeText(this, R.string.no_browser, Toast.LENGTH_LONG).show();
        }
    }

    /** Lets the person send the report to themselves, for example to look at it on a computer. */
    private void share() {
        if (savedUri == null) return;
        Intent send = new Intent(Intent.ACTION_SEND);
        send.setType("application/json");
        send.putExtra(Intent.EXTRA_STREAM, savedUri);
        send.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION);
        startActivity(Intent.createChooser(send, getString(R.string.share)));
    }
}
