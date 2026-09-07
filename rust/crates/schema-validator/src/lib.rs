//! EBC-X Domain Event and Evidence schema validation.

use serde::Deserialize;

#[derive(Debug, thiserror::Error)]
pub enum ValidationError {
    #[error("missing required field: {0}")]
    MissingField(String),
    #[error("invalid value for field {field}: {reason}")]
    InvalidValue { field: String, reason: String },
}

#[derive(Debug, Deserialize)]
pub struct EvidenceRecord {
    pub evidence_id: String,
    pub chain_id: String,
    pub sequence_no: u64,
    pub previous_evidence_hash: String,
    pub evidence_hash: String,
    pub payload: serde_json::Value,
    pub source_event_id: String,
    pub transaction_id: String,
    pub tenant_id: String,
    pub created_by: String,
    pub created_at: String,
}

pub fn validate(record: &EvidenceRecord) -> Result<(), ValidationError> {
    if record.evidence_id.is_empty() {
        return Err(ValidationError::MissingField("evidence_id".into()));
    }
    if record.chain_id.is_empty() {
        return Err(ValidationError::MissingField("chain_id".into()));
    }
    if record.evidence_hash.is_empty() {
        return Err(ValidationError::MissingField("evidence_hash".into()));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn valid_record() -> EvidenceRecord {
        EvidenceRecord {
            evidence_id: "evd-1".into(),
            chain_id: "chain-1".into(),
            sequence_no: 1,
            previous_evidence_hash: "prev".into(),
            evidence_hash: "hash".into(),
            payload: serde_json::json!({}),
            source_event_id: "evt-1".into(),
            transaction_id: "tx-1".into(),
            tenant_id: "t-1".into(),
            created_by: "u-1".into(),
            created_at: "2026-09-07T00:00:00Z".into(),
        }
    }

    #[test]
    fn test_valid_record() {
        assert!(validate(&valid_record()).is_ok());
    }

    #[test]
    fn test_missing_evidence_id() {
        let mut r = valid_record();
        r.evidence_id = String::new();
        assert!(validate(&r).is_err());
    }
}