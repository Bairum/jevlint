pub struct LocalizedTitle { pub text: String, pub locale: String }
pub struct StoredTitle { pub text: String, pub locale: String }
impl std::convert::From<LocalizedTitle> for StoredTitle {
    fn from(value: LocalizedTitle) -> StoredTitle {
        StoredTitle { text: value.text, locale: String::from("en") }
    }
}
