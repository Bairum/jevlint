use std::cmp::Ordering;

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
        match task.deadline_tick.cmp(&current.deadline_tick) {
            Ordering::Less => selected = Some(task),
            Ordering::Equal if task.ticket < current.ticket => selected = Some(task),
            _ => {}
        }
    }
    selected
}
