#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum LinkDiagnostic {
    Healthy,
    PacketLoss,
    Unreachable,
}

/// Encodes the diagnostic for a monitoring consumer: 0 = healthy,
/// 4 = packet-loss warning, 9 = unreachable error. These are status codes,
/// not comparison results or priorities to be ordered numerically.
pub fn diagnostic_status(diagnostic: LinkDiagnostic) -> u8 {
    match diagnostic {
        // Healthy branch: monitoring status 0 denotes normal operation.
        LinkDiagnostic::Healthy => 0,
        // Packet-loss branch: monitoring status 4 denotes a warning.
        LinkDiagnostic::PacketLoss => 4,
        // Unreachable branch: monitoring status 9 denotes an error.
        LinkDiagnostic::Unreachable => 9,
    }
}
