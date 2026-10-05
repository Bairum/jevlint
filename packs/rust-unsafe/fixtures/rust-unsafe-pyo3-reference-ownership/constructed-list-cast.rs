use pyo3::{ffi, prelude::*};
use pyo3::types::PyList;

pub fn empty_list<'py>(py: Python<'py>) -> PyResult<Bound<'py, PyList>> {
    // PyList_New returns a new reference to a Python list, or NULL with an exception.
    let pointer = unsafe { ffi::PyList_New(0) };
    let value = unsafe { Bound::from_owned_ptr_or_err(py, pointer) }?;
    Ok(unsafe { value.cast_into_unchecked::<PyList>() })
}
