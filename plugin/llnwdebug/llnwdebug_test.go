package llnwdebug

import (
	"testing"
	"time"

	"github.com/go-test/deep"
)

func TestLLNWDebug_recordResolver(t *testing.T) {
	type req struct {
		resolver, edns0Subnet, qtype string
	}
	tests := []struct {
		name        string
		requests    []req
		expectedLog []RequestInfo
	}{
		{"single add",
			[]req{
				{"10.0.0.1", "", "A"},
			},
			[]RequestInfo{
				{"10.0.0.1", "", "A"},
			},
		},
		{"multi add",
			[]req{
				{"10.0.0.1", "", "A"},
				{"10.0.0.1", "99.0.0.0/24", "A"},
				{"10.0.0.2", "", "A"},
				{"10.0.0.1", "", "A"},
				{"10.0.0.1", "99.0.0.0/24", "AAAA"},
			},
			[]RequestInfo{
				{"10.0.0.1", "", "A"},
				{"10.0.0.1", "99.0.0.0/24", "A"},
				{"10.0.0.2", "", "A"},
				{"10.0.0.1", "", "A"},
				{"10.0.0.1", "99.0.0.0/24", "AAAA"},
			},
		},
		{"abuse",
			[]req{
				{"10.0.0.1", "", "A"},
				{"10.0.0.2", "", "A"},
				{"10.0.0.3", "", "A"},
				{"10.0.0.4", "", "A"},
				{"10.0.0.5", "", "A"},
				{"10.0.0.6", "", "A"},
				{"10.0.0.7", "", "A"},
				{"10.0.0.8", "", "A"},
				{"10.0.0.9", "", "A"},
				{"10.0.0.10", "", "A"},
				{"10.0.0.11", "", "A"}, // Ignored
			},
			[]RequestInfo{
				{"10.0.0.1", "", "A"},
				{"10.0.0.2", "", "A"},
				{"10.0.0.3", "", "A"},
				{"10.0.0.4", "", "A"},
				{"10.0.0.5", "", "A"},
				{"10.0.0.6", "", "A"},
				{"10.0.0.7", "", "A"},
				{"10.0.0.8", "", "A"},
				{"10.0.0.9", "", "A"},
				{"10.0.0.10", "", "A"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ld := NewLLNWDebug(nil, nil)
			for _, req := range tt.requests {
				ld.recordResolver(tt.name, req.resolver, req.edns0Subnet, req.qtype)
			}
			actual, ok := ld.dnsRequests[tt.name]
			if !ok {
				t.Fatal("empty request log")
			}
			if actual.lastUpdate.IsZero() {
				t.Error("missing log timestamp")
			}
			if diff := deep.Equal(tt.expectedLog, actual.log); len(diff) > 0 {
				t.Errorf("Expected log %v, got %v. Diff %v\n", tt.expectedLog, actual.log, diff)
			}
		})
	}
}

func TestLLNWDebug_Cleanup(t *testing.T) {
	ld := NewLLNWDebug(nil, nil)
	ld.recordResolver("basic", "10.0.0.1", "", "A")

	rm, total := ld.Cleanup(time.Now())
	if rm != 1 || total != 1 {
		t.Errorf("Expected 1/1 cleanup, got %d/%d", rm, total)
	}
	if len(ld.dnsRequests) > 0 {
		t.Errorf("request not removed")
	}
}
