#[derive(Clone, Copy, Debug)]
pub struct Candidate {
    pub id: u64,
    pub assigned_rank: Option<u32>,
}

pub fn preferred_candidate(candidates: &[Candidate]) -> Option<&Candidate> {
    candidates
        .iter()
        .min_by_key(|candidate| candidate.assigned_rank.unwrap_or(999_999))
}
