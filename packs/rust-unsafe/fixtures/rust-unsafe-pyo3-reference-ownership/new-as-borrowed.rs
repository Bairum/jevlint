use pyo3::{ffi, prelude::*};

pub fn empty_list<'py>(py: Python<'py>) -> PyResult<Bound<'py, PyAny>> {
    // PyList_New returns a new reference, or NULL with an exception.
    let pointer = unsafe { ffi::PyList_New(0) };
    unsafe { Bound::from_borrowed_ptr_or_err(py, pointer) }
}
