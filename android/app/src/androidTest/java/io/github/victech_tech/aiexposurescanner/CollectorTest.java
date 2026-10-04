package io.github.victech_tech.aiexposurescanner;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertTrue;

import android.content.Context;
import android.util.Log;

import androidx.test.ext.junit.runners.AndroidJUnit4;
import androidx.test.platform.app.InstrumentationRegistry;

import org.junit.Test;
import org.junit.runner.RunWith;

/** Runs the real collector on an emulator (CI) or a phone. */
@RunWith(AndroidJUnit4.class)
public class CollectorTest {
    @Test
    public void collectsTheApps() {
        Context context = InstrumentationRegistry.getInstrumentation().getTargetContext();
        Report r = new Collector(context).collect("test");
        String json = r.toJson();
        Log.i("CollectorTest", json);

        assertFalse("some apps with an icon are listed", r.installedApps.isEmpty());
        for (Report.App a : r.installedApps) {
            assertFalse("the scanner does not list itself", a.id.equals(context.getPackageName()));
            assertFalse("every app has a name", a.name.isEmpty());
        }
        boolean keyboard = false;
        for (Report.Permission p : r.privacyPermissions) keyboard |= p.capability.equals("keyboard");
        assertTrue("the phone's keyboard is listed", keyboard);
        assertTrue(json.startsWith("{\n  \"schemaVersion\": 1,"));
        assertEquals(json.indexOf("\"family\": \"android\""), json.lastIndexOf("\"family\": \"android\""));
    }
}
