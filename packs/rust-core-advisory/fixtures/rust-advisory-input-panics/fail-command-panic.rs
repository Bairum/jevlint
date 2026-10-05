#[derive(Debug)]
pub enum Command {
    List,
    Fetch(String),
}

#[derive(Debug)]
pub struct CommandError;

pub fn parse_network_command(line: &str) -> Result<Command, CommandError> {
    match line.trim().split_once(' ') {
        Some(("FETCH", key)) if !key.is_empty() => Ok(Command::Fetch(key.to_owned())),
        None if line.trim() == "LIST" => Ok(Command::List),
        _ => panic!("unknown command syntax"),
    }
}
