package io.github.victech_tech.aiexposurescanner;

/**
 * A very small JSON writer (two-space indents), so the app needs no
 * libraries. Plain Java, unit tested.
 */
final class Json {
    private final StringBuilder out = new StringBuilder();
    private int depth = 0;
    /** True when the next item in the current object or array needs a comma first. */
    private final boolean[] needComma = new boolean[16];
    private boolean afterName = false;

    Json beginObject() {
        return open('{');
    }

    Json endObject() {
        return close('}');
    }

    Json beginArray() {
        return open('[');
    }

    Json endArray() {
        return close(']');
    }

    Json name(String name) {
        item();
        string(name);
        out.append(": ");
        afterName = true;
        return this;
    }

    Json value(String s) {
        item();
        string(s);
        return this;
    }

    Json value(int n) {
        item();
        out.append(n);
        return this;
    }

    Json value(boolean b) {
        item();
        out.append(b);
        return this;
    }

    @Override
    public String toString() {
        return out.toString() + "\n";
    }

    private Json open(char c) {
        item();
        out.append(c);
        depth++;
        needComma[depth] = false;
        return this;
    }

    private Json close(char c) {
        boolean empty = !needComma[depth];
        depth--;
        if (!empty) newline();
        out.append(c);
        return this;
    }

    /** Starts a new item: a comma and new line unless it follows a name. */
    private void item() {
        if (afterName) {
            afterName = false;
            return;
        }
        if (depth == 0) return;
        if (needComma[depth]) out.append(',');
        needComma[depth] = true;
        newline();
    }

    private void newline() {
        out.append('\n');
        for (int i = 0; i < depth; i++) out.append("  ");
    }

    private void string(String s) {
        out.append('"');
        for (int i = 0; i < s.length(); i++) {
            char c = s.charAt(i);
            switch (c) {
                case '"': out.append("\\\""); break;
                case '\\': out.append("\\\\"); break;
                case '\n': out.append("\\n"); break;
                case '\r': out.append("\\r"); break;
                case '\t': out.append("\\t"); break;
                default:
                    if (c < 0x20 || c == 0x2028 || c == 0x2029) {
                        out.append(String.format("\\u%04x", (int) c));
                    } else {
                        out.append(c);
                    }
            }
        }
        out.append('"');
    }
}
