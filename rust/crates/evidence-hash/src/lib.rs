use sha2::{Digest, Sha256};

/// Compute the chained hash for an Evidence record (TASK-H01).
///
/// H_n = SHA256(chain_id || sequence_no || previous_hash || payload || source_event_id || transaction_id)
pub fn compute_chain_hash(
    chain_id: &str,
    sequence_no: u64,
    previous_hash: &str,
    payload: &[u8],
    source_event_id: &str,
    transaction_id: &str,
) -> String {
    let mut hasher = Sha256::new();
    hasher.update(chain_id.as_bytes());
    hasher.update(sequence_no.to_be_bytes());
    hasher.update(previous_hash.as_bytes());
    hasher.update(payload);
    hasher.update(source_event_id.as_bytes());
    hasher.update(transaction_id.as_bytes());
    hex::encode(hasher.finalize())
}

/// Compute the genesis hash for the first Evidence in a chain.
pub fn compute_genesis_hash(chain_id: &str) -> String {
    let mut hasher = Sha256::new();
    hasher.update(b"genesis");
    hasher.update(chain_id.as_bytes());
    hex::encode(hasher.finalize())
}

/// Verify a chain of evidence hashes is continuous and unbroken.
pub fn verify_chain(hashes: &[(String, String)]) -> bool {
    if hashes.is_empty() {
        return true;
    }
    for window in hashes.windows(2) {
        if window[0].1 != window[1].0 {
            return false;
        }
    }
    true
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_chain_hash_deterministic() {
        let h1 = compute_chain_hash("chain-1", 1, "prev", b"payload", "evt-1", "tx-1");
        let h2 = compute_chain_hash("chain-1", 1, "prev", b"payload", "evt-1", "tx-1");
        assert_eq!(h1, h2);
    }

    #[test]
    fn test_tamper_detection() {
        let h1 = compute_chain_hash("chain-1", 1, "prev", b"payload", "evt-1", "tx-1");
        let h1_tampered = compute_chain_hash("chain-1", 1, "prev", b"TAMPERED", "evt-1", "tx-1");
        assert_ne!(h1, h1_tampered);
    }

    #[test]
    fn test_verify_chain() {
        let genesis = compute_genesis_hash("chain-1");
        let h1 = compute_chain_hash("chain-1", 1, &genesis, b"p1", "e1", "t1");
        let h2 = compute_chain_hash("chain-1", 2, &h1, b"p2", "e2", "t2");
        assert!(verify_chain(&[(genesis.clone(), h1.clone()), (h1, h2)]));
    }

    #[test]
    fn test_verify_broken_chain() {
        let chain = vec![
            ("a".to_string(), "b".to_string()),
            ("x".to_string(), "c".to_string()),
        ];
        assert!(!verify_chain(&chain));
    }
}