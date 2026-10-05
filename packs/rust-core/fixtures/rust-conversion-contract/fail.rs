/// Both fields are semantically significant: locale selects the language.
pub struct LocalizedTitle { pub text: String, pub locale: String }
/// A stored title must preserve both the text and the requested language.
pub struct StoredTitle { pub text: String, pub locale: String }
impl std::convert::From<LocalizedTitle> for StoredTitle {
    fn from(value: LocalizedTitle) -> StoredTitle {
        StoredTitle { text: value.text, locale: String::from("en") }
    }
}
