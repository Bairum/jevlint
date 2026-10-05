/// Write the payload to the child's input pipe. Return every local writing
/// and completion error, including when the child has closed its read end.
/// Success promises submission to the pipe, not child processing or durability.
pub fn write_child_input(input: std::process::ChildStdin) -> std::io::Result<()> {
    let mut writer = std::io::BufWriter::with_capacity(128, input);
    std::io::Write::write_all(&mut writer, b"Some bytes")?;
    std::io::Write::flush(&mut writer)?;
    Ok(())
}
