use pyo3::{ffi, prelude::*};

pub fn active_handle<'py>(py: Python<'py>) -> PyResult<Bound<'py, PyAny>> {
    // active_python_handle returns a borrowed reference to an interpreter-owned object,
    // valid while attached to Python; it returns NULL with an exception on failure.
    extern "C" {
        fn active_python_handle() -> *mut ffi::PyObject;
    }
    let pointer = unsafe { active_python_handle() };
    unsafe { Bound::from_owned_ptr_or_err(py, pointer) }
}
