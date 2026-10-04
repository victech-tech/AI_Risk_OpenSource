// The only build plugin is Google's Android plugin. The app itself uses no
// libraries at all: just the Android framework (see app/build.gradle.kts).
plugins {
    id("com.android.application") version "8.13.2" apply false
}
