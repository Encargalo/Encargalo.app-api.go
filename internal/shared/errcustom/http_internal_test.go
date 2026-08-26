package errcustom

import "testing"

func TestCodeMapCoversStatusMap(t *testing.T) {
	for sentinel := range statusMap {
		if _, ok := codeMap[sentinel]; !ok {
			t.Errorf("sentinel %q está en statusMap pero no tiene código v2 en codeMap", sentinel.Error())
		}
	}
}

func TestCodeMap_NoOrphanEntries(t *testing.T) {
	for sentinel := range codeMap {
		if _, ok := statusMap[sentinel]; !ok {
			t.Errorf("sentinel %q está en codeMap pero no en statusMap (entrada huérfana)", sentinel.Error())
		}
	}
}

func TestCodeMap_NoDuplicateCodes(t *testing.T) {
	seen := make(map[string]error, len(codeMap))
	for sentinel, code := range codeMap {
		if other, ok := seen[code]; ok {
			t.Errorf("código v2 %q duplicado entre %q y %q", code, other.Error(), sentinel.Error())
		}
		seen[code] = sentinel
	}
}

func TestCode_UnknownSentinelFallsBackToUnexpectedError(t *testing.T) {
	unknown := &customTestError{"nunca registrado en codeMap"}
	if got := Code(unknown); got != "unexpected_error" {
		t.Errorf("Code(unregistered) = %q, want %q", got, "unexpected_error")
	}
}

type customTestError struct{ msg string }

func (e *customTestError) Error() string { return e.msg }
