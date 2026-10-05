/// Both fields are semantically significant: locale selects the language.
pub struct LocalizedTitle { pub text: String, pub locale: String }
/// A stored title preserves text and language without changing their meaning.
pub struct StoredTitle { pub text: String, pub locale: String }
impl std::convert::From<LocalizedTitle> for StoredTitle {
    fn from(value: LocalizedTitle) -> StoredTitle {
        StoredTitle { text: value.text, locale: value.locale }
    }
}
