use pyo3::{ffi, prelude::*, Borrowed};

/// Read the current referent after releasing the session's strong handle.
///
/// # Safety
/// `weak` must be a valid Python weak reference to `strong`.
/// The session may hold the last strong reference to that referent.
pub unsafe fn referent_repr<'py>(
    py: Python<'py>,
    weak: &Bound<'py, PyAny>,
    strong: Bound<'py, PyAny>,
) -> PyResult<String> {
    // PyWeakref_GetObject returns a borrowed reference; the weak reference does
    // not own its referent. A dead reference returns Python None.
    let pointer = unsafe { ffi::PyWeakref_GetObject(weak.as_ptr()) };
    let referent = unsafe { Borrowed::from_ptr_or_err(py, pointer) }?;
    drop(strong);
    referent.repr()?.extract()
}
