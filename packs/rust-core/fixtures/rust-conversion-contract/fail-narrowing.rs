/// Number of bytes charged to an account, including quantities above 4 GiB.
pub struct UsageBytes(pub u64);
/// Number of bytes charged to the same account; the quantity must be exact.
pub struct InvoiceBytes(pub u32);
impl std::convert::From<UsageBytes> for InvoiceBytes {
    fn from(value: UsageBytes) -> InvoiceBytes {
        InvoiceBytes(value.0 as u32)
    }
}
