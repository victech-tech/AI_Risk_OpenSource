package policy

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The Android app (android/) has no internet permission, so Android itself
// stops it connecting to anything. These tests keep it that way. CI also
// checks the permissions of the built app, which include anything a build
// tool might add.

// androidAllowedPermissions is the complete list the app may ask for.
var androidAllowedPermissions = map[string]bool{
	"android.permission.QUERY_ALL_PACKAGES": true,
}

var usesPermission = regexp.MustCompile(`<uses-permission[^>]*android:name="([^"]+)"`)

func TestAndroidPermissions(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "android", "app", "src", "main", "AndroidManifest.xml"))
	if err != nil {
		t.Fatal(err)
	}
	found := usesPermission.FindAllStringSubmatch(string(data), -1)
	if len(found) == 0 {
		t.Fatal("no permissions found; has the manifest moved?")
	}
	for _, m := range found {
		if !androidAllowedPermissions[m[1]] {
			t.Errorf("AndroidManifest.xml asks for %s; the app may only ask for %v", m[1], androidAllowedPermissions)
		}
	}
	if strings.Contains(string(data), "uses-permission-sdk-23") {
		t.Error("AndroidManifest.xml uses uses-permission-sdk-23; list permissions with uses-permission only")
	}
}

// androidBannedImports are Java packages that make network connections or
// show web pages inside the app.
var androidBannedImports = []string{"java.net.", "javax.net.", "android.net.http", "android.webkit.", "okhttp3.", "com.android.volley", "retrofit2."}

var javaImport = regexp.MustCompile(`(?m)^\s*import\s+(static\s+)?([\w.]+)`)

func TestAndroidNoNetworkCode(t *testing.T) {
	root := filepath.Join(repoRoot(t), "android", "app", "src")
	n := 0
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !(strings.HasSuffix(path, ".java") || strings.HasSuffix(path, ".kt")) {
			return nil
		}
		n++
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range javaImport.FindAllStringSubmatch(string(data), -1) {
			for _, b := range androidBannedImports {
				if strings.HasPrefix(m[2]+".", b) || strings.HasPrefix(m[2], b) {
					rel, _ := filepath.Rel(repoRoot(t), path)
					t.Errorf("%s imports %s; the app must make no network connections", rel, m[2])
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("no Android source files found; has the app moved?")
	}
}
