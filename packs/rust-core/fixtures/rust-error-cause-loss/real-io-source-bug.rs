#[derive(Debug)]
pub enum CertificateFailure {
    UnknownIssuer,
}

impl std::fmt::Display for CertificateFailure {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter.write_str("certificate issuer is not trusted")
    }
}

impl std::error::Error for CertificateFailure {}

#[derive(Debug)]
pub struct TransportFailure {
    cause: CertificateFailure,
}

impl TransportFailure {
    pub fn from_certificate(cause: CertificateFailure) -> Self {
        Self { cause }
    }
}

impl std::fmt::Display for TransportFailure {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter.write_str("TLS connection failed")
    }
}

impl std::error::Error for TransportFailure {
    /// Expose the immediate certificate validation error through the standard
    /// source chain. Credential clients downcast the cause to CertificateFailure
    /// and provide trust-store repair instructions for UnknownIssuer.
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        std::error::Error::source(&self.cause)
    }
}
