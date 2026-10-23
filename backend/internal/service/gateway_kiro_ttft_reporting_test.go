package service

import (
	"testing"
)

func TestKiroTTFTReporting(t *testing.T) {
	var p kiroTTFTReporting
	for _, malformed := range []string{": kiro-first-token-cap-ms=abc", ": kiro-first-token-cap-ms=-1", ": kiro-first-token-cap-ms=2999"} {
		if p.observe(malformed) {
			t.Fatal("invalid marker accepted")
		}
	}
	if p.reported(26090) != 26090 {
		t.Fatal("unmarked upstream must retain actual timing")
	}
	if p.observe(": keepalive") {
		t.Fatal("keepalive is not a header marker")
	}
	if !p.observe(": kiro-first-token-cap-ms=4800") {
		t.Fatal("missing marker")
	}
	if got := p.reported(26090); got != 4800 {
		t.Fatalf("got %d, want 4800", got)
	}
	if got := p.reported(2500); got != 2500 {
		t.Fatalf("fast response changed to %d", got)
	}
	p.observe(": kiro-first-token-cap-ms=15000")
	if p.reported(26090) != 4800 {
		t.Fatal("retry must not renew the reporting cap")
	}
}
