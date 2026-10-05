// Legacy ranking policy gives unranked candidates this key. Assigned ranks
// equal to or above it retain the existing tie/order behavior.
const UNRANKED_SORT_KEY: u32 = 999_999;

#[derive(Clone, Copy, Debug)]
pub struct Candidate {
    pub id: u64,
    pub assigned_rank: Option<u32>,
}

pub fn preferred_candidate(candidates: &[Candidate]) -> Option<&Candidate> {
    candidates
        .iter()
        .min_by_key(|candidate| candidate.assigned_rank.unwrap_or(UNRANKED_SORT_KEY))
}
