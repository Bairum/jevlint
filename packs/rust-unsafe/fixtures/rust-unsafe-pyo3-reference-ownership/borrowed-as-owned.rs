use pyo3::{ffi, prelude::*};
use pyo3::types::PyList;

pub fn first_item<'py>(py: Python<'py>, list: &Bound<'py, PyList>) -> PyResult<Bound<'py, PyAny>> {
    // PyList_GetItem returns a borrowed reference, or NULL with an exception.
    let pointer = unsafe { ffi::PyList_GetItem(list.as_ptr(), 0) };
    unsafe { Bound::from_owned_ptr_or_err(py, pointer) }
}
