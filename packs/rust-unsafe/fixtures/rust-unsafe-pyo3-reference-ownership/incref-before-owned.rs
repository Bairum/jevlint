use pyo3::{ffi, prelude::*};

pub fn duplicate<'py>(py: Python<'py>, value: &Bound<'py, PyAny>) -> Bound<'py, PyAny> {
    let pointer = value.as_ptr();
    unsafe { ffi::Py_INCREF(pointer) };
    unsafe { Bound::from_owned_ptr(py, pointer) }
}
