package collect

import (
	"runtime"

	"github.com/victech-tech/AI_Risk_OpenSource/internal/report"
)

func osInfoGeneric() report.OS {
	return report.OS{Family: runtime.GOOS, Arch: runtime.GOARCH}
}
