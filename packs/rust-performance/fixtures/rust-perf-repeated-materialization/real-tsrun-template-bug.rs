use std::collections::BTreeMap;

#[derive(Clone)]
pub enum RegisterValue {
    Text(String),
    Object(BTreeMap<String, String>),
}

/// Renders template fragments from the worker's register window.
///
/// Bulk rendering jobs contain at least 100,000 fragments across their template
/// windows. Object registers contain immutable named rendering results, with
/// `toString` selecting their custom representation. Text registers are emitted
/// directly. A missing custom representation contributes an empty fragment.
/// Registers remain unchanged throughout rendering, and no fragment consumer
/// retains a register or a property key. Only the concatenated output is owned
/// by the caller. Property names are case sensitive and preserve Unicode bytes.
pub fn render_template(registers: &[RegisterValue]) -> String {
    let mut result = String::new();
    for index in 0..registers.len() {
        let value = registers[index].clone();
        let fragment: &str = match &value {
            RegisterValue::Object(properties) => {
                let to_string_key: String = "toString".to_owned();
                properties.get(&to_string_key).map(String::as_str).unwrap_or("")
            }
            RegisterValue::Text(text) => text.as_str(),
        };
        result.push_str(fragment);
    }
    result
}
