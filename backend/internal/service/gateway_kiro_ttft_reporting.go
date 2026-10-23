package service

import (
	"strconv"
	"strings"
)

// Only Kiro's explicit header marker opts a stream into this reporting policy.
// It never counts as model content, creates a sample, or changes the stream.
type kiroTTFTReporting struct{ capMs *int }

func (p *kiroTTFTReporting) observe(line string) bool {
	value, ok := strings.CutPrefix(strings.TrimSpace(line), ": kiro-first-token-cap-ms=")
	if !ok {
		return false
	}
	capMs, err := strconv.Atoi(value)
	if err != nil || capMs < 3000 {
		return false
	}
	if p.capMs == nil {
		p.capMs = &capMs
	}
	return true
}

func (p *kiroTTFTReporting) reported(actualMs int) int {
	if p.capMs != nil && actualMs > *p.capMs {
		return *p.capMs
	}
	return actualMs
}
