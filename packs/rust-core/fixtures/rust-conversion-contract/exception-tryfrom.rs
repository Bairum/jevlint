pub struct Port(pub u16);
impl std::convert::TryFrom<u32> for Port {
    type Error = std::num::TryFromIntError;
    fn try_from(value: u32) -> std::result::Result<Port, Self::Error> {
        <u16 as std::convert::TryFrom<u32>>::try_from(value).map(Port)
    }
}
