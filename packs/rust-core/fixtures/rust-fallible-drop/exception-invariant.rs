/// MustFinish requires the owner to call finish before release. Violations of
/// this programmer-owned protocol deliberately trigger a fail-fast assertion.
pub struct MustFinish { finished: bool }
impl MustFinish {
    pub fn new() -> MustFinish { MustFinish { finished: false } }
    /// MustFinish marks the programmer-owned completion obligation satisfied.
    pub fn finish(mut self) { self.finished = true; }
}
impl std::ops::Drop for MustFinish {
    /// MustFinish intentionally fails fast for a violated programmer invariant.
    fn drop(&mut self) {
        assert!(self.finished, "programmer must explicitly finish MustFinish");
    }
}
