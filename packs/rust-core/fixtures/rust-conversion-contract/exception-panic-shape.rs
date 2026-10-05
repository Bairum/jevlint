pub struct Marker;
impl std::convert::From<String> for Marker {
    fn from(_value: String) -> Marker {
        panic!("conversion is not available")
    }
}
