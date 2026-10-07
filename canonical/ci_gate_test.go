package canonical_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestCIGrepGate_NoLocalSportIDMaps rejects any hardcoded legacy sport IDs (such as SportIDCricket = 66
// or SportIDIceHockey = 2) defined outside contracts/canonical.
func TestCIGrepGate_NoLocalSportIDMaps(t *testing.T) {
	workspaceRoot := filepath.Clean("../..")

	forbiddenPatterns := []*regexp.Regexp{
		regexp.MustCompile(`SportIDCricket\s*=\s*66`),
		regexp.MustCompile(`SportIDIceHockey\s*=\s*2\b`),
		regexp.MustCompile(`SportIDBasketball\s*=\s*3\b`),
		regexp.MustCompile(`SportIDTennis\s*=\s*4\b`),
		regexp.MustCompile(`SportIDVolleyball\s*=\s*6\b`),
		regexp.MustCompile(`SportIDMMA\s*=\s*9\b`),
		regexp.MustCompile(`SportIDTableTennis\s*=\s*10\b`),
	}

	servicesToScan := []string{
		"prematch-service",
		"live-service",
		"settlement-service",
		"bets-service",
	}

	for _, svc := range servicesToScan {
		dir := filepath.Join(workspaceRoot, svc)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if info.Name() == "vendor" || info.Name() == "node_modules" || info.Name() == ".git" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(info.Name(), ".go") {
				return nil
			}

			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}

			str := string(content)
			for _, pat := range forbiddenPatterns {
				if pat.MatchString(str) {
					t.Errorf("CI GREP GATE VIOLATION: Forbidden legacy sport ID map found in %s matching %q", path, pat.String())
				}
			}
			return nil
		})

		if err != nil {
			t.Fatalf("Failed to scan directory %s: %v", dir, err)
		}
	}
}
