use std::io::{self, BufRead, Write};

pub fn print_matching_records(prefix: &str) -> io::Result<usize> {
    let input = io::stdin();
    let mut output = io::stdout().lock();
    let mut matches = 0;
    for line in input.lock().lines() {
        let line = line?;
        if line.starts_with(prefix) {
            writeln!(output, "{line}")?;
            matches += 1;
        }
    }
    output.flush()?;
    Ok(matches)
}

