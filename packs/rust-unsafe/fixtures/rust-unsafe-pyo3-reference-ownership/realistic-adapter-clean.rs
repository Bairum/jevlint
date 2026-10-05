use pyo3::{ffi, prelude::*};
use pyo3::types::{PyList, PyListMethods};

pub struct TableSnapshot {
    pub title: String,
    pub columns: Vec<String>,
    pub rows: Vec<Vec<String>>,
}

pub fn new_rows<'py>(py: Python<'py>) -> PyResult<Bound<'py, PyAny>> {
    // PyList_New returns a new reference, or NULL with an exception.
    let pointer = unsafe { ffi::PyList_New(0) };
    unsafe { Bound::from_owned_ptr_or_err(py, pointer) }
}

pub fn first_cell<'py>(
    py: Python<'py>,
    row: &Bound<'py, PyList>,
) -> PyResult<Bound<'py, PyAny>> {
    // PyList_GetItem returns a borrowed reference, or NULL with an exception.
    let pointer = unsafe { ffi::PyList_GetItem(row.as_ptr(), 0) };
    unsafe { Bound::from_borrowed_ptr_or_err(py, pointer) }
}

pub fn cell_at<'py>(
    py: Python<'py>,
    row: &Bound<'py, PyList>,
    index: isize,
) -> PyResult<Bound<'py, PyAny>> {
    // PyList_GetItem returns a borrowed reference, or NULL with an exception.
    let pointer = unsafe { ffi::PyList_GetItem(row.as_ptr(), index) };
    unsafe { Bound::from_borrowed_ptr_or_err(py, pointer) }
}

pub fn list_width(value: &Bound<'_, PyAny>) -> Option<usize> {
    if unsafe { ffi::PyList_Check(value.as_ptr()) } == 0 {
        return None;
    }
    let list = unsafe { value.cast_unchecked::<PyList>() };
    Some(list.len())
}

pub fn title_text(value: &Bound<'_, PyAny>) -> PyResult<String> {
    value.str()?.extract()
}

pub fn columns_text(columns: &Bound<'_, PyList>) -> PyResult<Vec<String>> {
    let mut result = Vec::with_capacity(columns.len());
    for column in columns.iter() {
        result.push(column.str()?.extract()?);
    }
    Ok(result)
}

pub fn row_text(row: &Bound<'_, PyList>) -> PyResult<Vec<String>> {
    let mut result = Vec::with_capacity(row.len());
    for cell in row.iter() {
        result.push(cell.str()?.extract()?);
    }
    Ok(result)
}

pub fn snapshot(
    title: &Bound<'_, PyAny>,
    columns: &Bound<'_, PyList>,
    rows: &Bound<'_, PyList>,
) -> PyResult<TableSnapshot> {
    let mut output = Vec::with_capacity(rows.len());
    for row in rows.iter() {
        let row = row.cast::<PyList>()?;
        output.push(row_text(row)?);
    }
    Ok(TableSnapshot {
        title: title_text(title)?,
        columns: columns_text(columns)?,
        rows: output,
    })
}

pub fn rectangular(table: &TableSnapshot) -> bool {
    table.rows.iter().all(|row| row.len() == table.columns.len())
}

pub fn render(table: &TableSnapshot) -> String {
    let mut output = String::new();
    output.push_str(&table.title);
    output.push('\n');
    output.push_str(&table.columns.join(" | "));
    output.push('\n');
    for row in &table.rows {
        output.push_str(&row.join(" | "));
        output.push('\n');
    }
    output
}
