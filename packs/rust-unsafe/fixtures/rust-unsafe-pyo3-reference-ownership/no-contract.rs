use pyo3::{ffi, prelude::*};

pub fn active_handle<'py>(py: Python<'py>) -> PyResult<Bound<'py, PyAny>> {
    extern "C" {
        fn active_python_handle() -> *mut ffi::PyObject;
    }
    let pointer = unsafe { active_python_handle() };
    unsafe { Bound::from_owned_ptr_or_err(py, pointer) }
}
