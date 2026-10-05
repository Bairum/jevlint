#[derive(Clone, Copy, Debug)]
pub struct Task {
    pub deadline_tick: u64,
    pub ticket: u64,
}

pub fn next_task(tasks: &[Task]) -> Option<&Task> {
    let mut selected: Option<&Task> = None;
    for task in tasks {
        let Some(current) = selected else {
            selected = Some(task);
            continue;
        };
        let comparison = if task.deadline_tick < current.deadline_tick {
            10
        } else if task.deadline_tick == current.deadline_tick {
            20
        } else {
            30
        };
        match comparison {
            10 => selected = Some(task),
            20 if task.ticket < current.ticket => selected = Some(task),
            _ => {}
        }
    }
    selected
}
