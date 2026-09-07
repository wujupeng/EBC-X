package evidence

import (
	"encoding/json"
	"fmt"
	"testing"
)

func makePayload(s string) []byte {
	b, _ := json.Marshal(map[string]any{"data": s})
	return b
}

func buildChain(t *testing.T, chainID string, count int) []Record {
	t.Helper()
	records := make([]Record, 0, count)
	prev := GenesisHash(chainID)
	for i := int64(1); i <= int64(count); i++ {
		payload := makePayload(fmt.Sprintf("payload-%d", i))
		hash := ChainHash(chainID, i, prev, payload, fmt.Sprintf("evt-%d", i), fmt.Sprintf("tx-%d", i))
		records = append(records, Record{
			EvidenceID:           fmt.Sprintf("evd-%d", i),
			ChainID:              chainID,
			SequenceNo:           i,
			PreviousEvidenceHash: prev,
			EvidenceHash:         hash,
			EvidenceType:         EvidenceMandatory,
			Payload:              map[string]any{"data": fmt.Sprintf("payload-%d", i)},
			SourceEventID:        fmt.Sprintf("evt-%d", i),
			TransactionID:        fmt.Sprintf("tx-%d", i),
		})
		prev = hash
	}
	return records
}

func TestGenesisHashDeterministic(t *testing.T) {
	h1 := GenesisHash("chain-1")
	h2 := GenesisHash("chain-1")
	if h1 != h2 {
		t.Fatal("genesis hash not deterministic")
	}
	if len(h1) != 64 {
		t.Fatalf("genesis hash length = %d, want 64", len(h1))
	}
}

func TestChainHashDeterministic(t *testing.T) {
	h1 := ChainHash("c", 1, "prev", []byte("p"), "e", "t")
	h2 := ChainHash("c", 1, "prev", []byte("p"), "e", "t")
	if h1 != h2 {
		t.Fatal("chain hash not deterministic")
	}
}

func TestVerifyChainValid(t *testing.T) {
	records := buildChain(t, "chain-1", 5)
	result := VerifyChain(records)
	if !result.Valid {
		t.Fatalf("valid chain rejected: %s", result.Reason)
	}
	if result.TotalNodes != 5 {
		t.Fatalf("total nodes = %d, want 5", result.TotalNodes)
	}
}

func TestVerifyChainTampered(t *testing.T) {
	records := buildChain(t, "chain-1", 3)
	records[1].EvidenceHash = "tampered"
	result := VerifyChain(records)
	if result.Valid {
		t.Fatal("tampered chain accepted as valid")
	}
	if result.BrokenAt != 3 {
		t.Fatalf("broken at = %d, want 3 (tampering record[1].hash breaks chain at record[2])", result.BrokenAt)
	}
}

func TestDetectTamperClean(t *testing.T) {
	records := buildChain(t, "chain-1", 3)
	result := DetectTamper(records)
	if result.Detected {
		t.Fatalf("false positive tamper: %s", result.Reason)
	}
}

func TestDetectTamperModified(t *testing.T) {
	records := buildChain(t, "chain-1", 3)
	records[1].Payload = map[string]any{"data": "TAMPERED"}
	result := DetectTamper(records)
	if !result.Detected {
		t.Fatal("tamper not detected after payload modification")
	}
	if result.EvidenceID != "evd-2" {
		t.Fatalf("tamper evidence id = %s, want evd-2", result.EvidenceID)
	}
}

func TestDetectTamperHashForged(t *testing.T) {
	records := buildChain(t, "chain-1", 3)
	records[1].EvidenceHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	result := DetectTamper(records)
	if !result.Detected {
		t.Fatal("forged hash not detected")
	}
}

func TestGapDetection(t *testing.T) {
	records := buildChain(t, "chain-1", 5)
	records[2].SequenceNo = 5
	result := VerifyChain(records)
	if result.Valid {
		t.Fatal("gap not detected")
	}
	if !contains(result.Reason, "gap") {
		t.Fatalf("gap not reported: %s", result.Reason)
	}
}

func TestForkDetection(t *testing.T) {
	records := buildChain(t, "chain-1", 3)
	records[1].ChainID = "chain-2"
	result := VerifyChain(records)
	if result.Valid {
		t.Fatal("fork not detected")
	}
	if !contains(result.Reason, "fork") {
		t.Fatalf("fork not reported: %s", result.Reason)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
