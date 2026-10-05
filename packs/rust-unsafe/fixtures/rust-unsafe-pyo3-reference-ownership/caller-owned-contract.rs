use pyo3::{ffi, prelude::*};

/// Adopt the output of an extension's object factory.
///
/// # Safety
/// `pointer` must be a valid new Python reference for this interpreter, with
/// ownership transferred to this function; NULL must have a Python error set.
pub unsafe fn adopt<'py>(py: Python<'py>, pointer: *mut ffi::PyObject) -> PyResult<Bound<'py, PyAny>> {
    unsafe { Bound::from_owned_ptr_or_err(py, pointer) }
}
