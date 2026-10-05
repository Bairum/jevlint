use std::fmt;
use std::io;
use std::path::Path;

#[derive(Debug, PartialEq, Eq)]
pub enum FormatError {
    WidthTooLarge,
    MissingDirective,
    UnknownDirective(char),
}

impl fmt::Display for FormatError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::WidthTooLarge => formatter.write_str("field width is too large"),
            Self::MissingDirective => formatter.write_str("missing format directive"),
            Self::UnknownDirective(code) => write!(formatter, "unknown directive: {code}"),
        }
    }
}

impl std::error::Error for FormatError {}

#[derive(Debug)]
enum Field {
    Path,
    Size,
}

#[derive(Debug)]
enum Segment {
    Literal(String),
    Field { field: Field, width: usize },
}

#[derive(Debug)]
pub struct FileRecord {
    path: String,
    size: u64,
}

pub fn inspect_file(path: &Path) -> io::Result<FileRecord> {
    let metadata = path.metadata()?;
    Ok(FileRecord {
        path: path.display().to_string(),
        size: metadata.len(),
    })
}

pub fn parse_format_width(start: &str) -> Result<(Option<usize>, usize), FormatError> {
    let digits = start.bytes().take_while(u8::is_ascii_digit).count();
    if digits == 0 {
        return Ok((None, 0));
    }
    let width = start[..digits].parse::<usize>().map_err(|_| FormatError::WidthTooLarge)?;
    if width > u16::MAX as usize {
        return Err(FormatError::WidthTooLarge);
    }
    Ok((Some(width), digits))
}

#[derive(Debug)]
pub struct FormatPlan {
    segments: Vec<Segment>,
}

impl FormatPlan {
    pub fn compile(format: &str) -> Result<Self, FormatError> {
        let mut segments = Vec::new();
        let mut literal = String::new();
        let mut cursor = 0;
        while cursor < format.len() {
            let current = format[cursor..].chars().next().ok_or(FormatError::MissingDirective)?;
            cursor += current.len_utf8();
            if current != '%' {
                literal.push(current);
                continue;
            }
            if !literal.is_empty() {
                segments.push(Segment::Literal(std::mem::take(&mut literal)));
            }
            let (width, consumed) = parse_format_width(&format[cursor..])?;
            cursor += consumed;
            let directive = format[cursor..].chars().next().ok_or(FormatError::MissingDirective)?;
            cursor += directive.len_utf8();
            let field = match directive {
                '%' if width.is_none() => {
                    literal.push('%');
                    continue;
                }
                'p' => Field::Path,
                's' => Field::Size,
                other => return Err(FormatError::UnknownDirective(other)),
            };
            segments.push(Segment::Field {
                field,
                width: width.unwrap_or(0),
            });
        }
        if !literal.is_empty() {
            segments.push(Segment::Literal(literal));
        }
        Ok(Self { segments })
    }

    pub fn render(&self, record: &FileRecord) -> String {
        let mut output = String::new();
        for segment in &self.segments {
            match segment {
                Segment::Literal(text) => output.push_str(text),
                Segment::Field { field: Field::Path, width } => {
                    append_padded(&mut output, &record.path, *width);
                }
                Segment::Field { field: Field::Size, width } => {
                    append_padded(&mut output, &record.size.to_string(), *width);
                }
            }
        }
        output
    }
}

fn append_padded(output: &mut String, value: &str, width: usize) {
    let padding = width.saturating_sub(value.chars().count());
    for _ in 0..padding {
        output.push(' ');
    }
    output.push_str(value);
}
