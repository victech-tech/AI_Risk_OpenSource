package collect

// StartupSwitchedOff reports whether a StartupApproved registry value means
// "switched off". Windows writes it when someone uses the switch in
// Settings > Apps > Startup or Task Manager. The first byte is even (2 or 6)
// for on and odd (3 or 7) for off; the rest is the time it was changed.
func StartupSwitchedOff(value []byte) bool {
	return len(value) > 0 && value[0]&1 == 1
}
