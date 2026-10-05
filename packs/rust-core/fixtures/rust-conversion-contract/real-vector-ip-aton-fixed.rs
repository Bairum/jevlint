/// IPv4 address as the integer a*2^24 + b*2^16 + c*2^8 + d for octets a.b.c.d.
/// This numeric value is independent of the host architecture.
pub struct IpNumber(pub u32);

impl std::convert::From<std::net::Ipv4Addr> for IpNumber {
    fn from(address: std::net::Ipv4Addr) -> IpNumber {
        IpNumber(u32::from(address))
    }
}
