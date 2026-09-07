package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

func computeHash(data ...[]byte) string {
	h := sha256.New()
	for _, d := range data {
		h.Write(d)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func GenesisHash(chainID string) string {
	return computeHash([]byte("genesis"), []byte(chainID))
}

func ChainHash(chainID string, sequenceNo int64, previousHash string, payload []byte, sourceEventID, transactionID string) string {
	return computeHash(
		[]byte(chainID),
		[]byte(fmt.Sprintf("%d", sequenceNo)),
		[]byte(previousHash),
		payload,
		[]byte(sourceEventID),
		[]byte(transactionID),
	)
}

func VerifyChain(records []Record) ChainVerificationResult {
	if len(records) == 0 {
		return ChainVerificationResult{Valid: true, TotalNodes: 0}
	}
	for i := 1; i < len(records); i++ {
		if records[i].PreviousEvidenceHash != records[i-1].EvidenceHash {
			return ChainVerificationResult{
				Valid:      false,
				BrokenAt:   records[i].SequenceNo,
				Reason:     fmt.Sprintf("hash chain broken at sequence %d: prev_hash mismatch", records[i].SequenceNo),
				TotalNodes: len(records),
			}
		}
		if records[i].ChainID != records[i-1].ChainID {
			return ChainVerificationResult{
				Valid:      false,
				BrokenAt:   records[i].SequenceNo,
				Reason:     fmt.Sprintf("chain_id changed at sequence %d (fork detected)", records[i].SequenceNo),
				TotalNodes: len(records),
			}
		}
		if records[i].SequenceNo != records[i-1].SequenceNo+1 {
			return ChainVerificationResult{
				Valid:      false,
				BrokenAt:   records[i].SequenceNo,
				Reason:     fmt.Sprintf("gap detected: expected sequence %d, got %d", records[i-1].SequenceNo+1, records[i].SequenceNo),
				TotalNodes: len(records),
			}
		}
	}
	return ChainVerificationResult{Valid: true, TotalNodes: len(records)}
}

func DetectTamper(records []Record) TamperResult {
	for i := range records {
		payloadBytes, _ := json.Marshal(records[i].Payload)
		expected := ChainHash(
			records[i].ChainID,
			records[i].SequenceNo,
			records[i].PreviousEvidenceHash,
			payloadBytes,
			records[i].SourceEventID,
			records[i].TransactionID,
		)
		if !strings.EqualFold(expected, records[i].EvidenceHash) {
			return TamperResult{
				Detected:   true,
				EvidenceID: records[i].EvidenceID,
				Reason:     fmt.Sprintf("hash mismatch for evidence %s: expected %s, got %s", records[i].EvidenceID, expected, records[i].EvidenceHash),
			}
		}
	}
	return TamperResult{Detected: false}
}
