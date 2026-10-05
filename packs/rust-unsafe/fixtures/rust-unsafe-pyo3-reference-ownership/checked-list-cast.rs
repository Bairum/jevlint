use pyo3::{ffi, prelude::*};
use pyo3::types::{PyList, PyListMethods};

pub fn value_count(value: &Bound<'_, PyAny>) -> Option<usize> {
    if unsafe { ffi::PyList_Check(value.as_ptr()) } == 0 {
        return None;
    }
    let list = unsafe { value.cast_unchecked::<PyList>() };
    Some(list.len())
}
