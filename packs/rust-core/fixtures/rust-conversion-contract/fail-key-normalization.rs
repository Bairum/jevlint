/// Tenant keys are case-sensitive: "Acme" and "acme" identify different tenants.
pub struct TenantKey(pub String);
/// Stored keys identify the same tenant with exactly the original spelling.
pub struct StoredTenantKey(pub String);
impl std::convert::From<TenantKey> for StoredTenantKey {
    fn from(value: TenantKey) -> StoredTenantKey {
        StoredTenantKey(value.0.to_ascii_lowercase())
    }
}
