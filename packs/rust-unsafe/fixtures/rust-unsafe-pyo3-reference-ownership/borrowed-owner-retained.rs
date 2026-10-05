use pyo3::{prelude::*, Borrowed};
use pyo3::types::{PyList, PyListMethods};

pub fn list_size(py: Python<'_>, list: &Bound<'_, PyList>) -> PyResult<usize> {
    let borrowed = unsafe { Borrowed::from_ptr(py, list.as_ptr()) };
    let view = borrowed.cast::<PyList>()?;
    Ok(view.len())
}
