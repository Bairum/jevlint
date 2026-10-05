use pyo3::prelude::*;
use pyo3::types::{PyInt, PyList, PyListMethods};

pub fn value_count(py: Python<'_>, value: i64) -> usize {
    let object = PyInt::new(py, value).into_any();
    let list = unsafe { object.cast_unchecked::<PyList>() };
    list.len()
}
