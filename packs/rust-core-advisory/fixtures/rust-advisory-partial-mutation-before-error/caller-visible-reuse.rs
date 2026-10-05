pub struct ReadingList {
    pub titles: Vec<String>,
    pub selected: usize,
}

impl ReadingList {
    pub fn replace(&mut self, titles: Vec<String>, selected: &str) -> Result<(), &'static str> {
        self.titles.clear();
        let selected = selected.parse::<usize>().map_err(|_| "invalid selection")?;
        if selected >= titles.len() {
            return Err("selection out of range");
        }
        self.titles = titles;
        self.selected = selected;
        Ok(())
    }

    pub fn edit_and_current(&mut self, titles: Vec<String>, selected: &str) -> &str {
        if let Err(error) = self.replace(titles, selected) {
            eprintln!("Could not replace reading list: {error}");
        }
        &self.titles[self.selected]
    }
}
