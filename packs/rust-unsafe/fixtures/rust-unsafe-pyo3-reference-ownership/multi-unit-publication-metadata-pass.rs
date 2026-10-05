use pyo3::{exceptions::PyValueError, ffi, prelude::*};
use pyo3::types::{PyDict, PyList, PyListMethods};

pub struct PublicationSummary {
    pub title: String,
    pub record_count: usize,
}

pub fn empty_metadata<'py>(py: Python<'py>) -> PyResult<Bound<'py, PyAny>> {
    // PyDict_New returns a new reference, or NULL with an exception set.
    let pointer = unsafe { ffi::PyDict_New() };
    unsafe { Bound::from_owned_ptr_or_err(py, pointer) }
}

pub fn record_at<'py>(
    py: Python<'py>,
    records: &Bound<'py, PyList>,
    index: isize,
) -> PyResult<Bound<'py, PyAny>> {
    // PyList_GetItem returns a borrowed reference; NULL signals an exception.
    let pointer = unsafe { ffi::PyList_GetItem(records.as_ptr(), index) };
    unsafe { Bound::from_borrowed_ptr_or_err(py, pointer) }
}

pub fn prepare_publication<'py>(
    py: Python<'py>,
    title: &str,
    records: &Bound<'py, PyList>,
    metadata: &Bound<'py, PyDict>,
) -> PyResult<Bound<'py, PyAny>> {
    let title_length = isize::try_from(title.len())
        .map_err(|_| PyValueError::new_err("title exceeds Python string length"))?;

    // PyDict_New returns a new reference or NULL with an exception set.
    let dictionary_pointer = unsafe { ffi::PyDict_New() };
    let publication = unsafe { Bound::from_owned_ptr_or_err(py, dictionary_pointer) }?;

    // PyList_New(0) returns a new reference to an initialized empty list,
    // or NULL with an exception set. No element slots need filling at length zero.
    let list_pointer = unsafe { ffi::PyList_New(0) };
    let published_records = unsafe { Bound::from_owned_ptr_or_err(py, list_pointer) }?;

    // PyUnicode_FromStringAndSize reads exactly title_length UTF-8 bytes and
    // returns a new reference, or NULL with an exception set.
    let title_pointer = unsafe {
        ffi::PyUnicode_FromStringAndSize(title.as_ptr().cast(), title_length)
    };
    let title_object = unsafe { Bound::from_owned_ptr_or_err(py, title_pointer) }?;

    // PyDict_SetItemString does not steal the value reference and reports -1
    // with an exception on failure. Its key argument is NUL terminated.
    let title_status = unsafe {
        ffi::PyDict_SetItemString(
            publication.as_ptr(),
            b"title\0".as_ptr().cast(),
            title_object.as_ptr(),
        )
    };
    if title_status != 0 {
        return Err(PyErr::fetch(py));
    }

    // PyDict_GetItemString returns a borrowed reference or NULL for an absent
    // key. The metadata dictionary remains owned throughout this lookup.
    let category_pointer = unsafe {
        ffi::PyDict_GetItemString(metadata.as_ptr(), b"category\0".as_ptr().cast())
    };
    if !category_pointer.is_null() {
        let category = unsafe { Bound::from_borrowed_ptr(py, category_pointer) };
        let category_status = unsafe {
            ffi::PyDict_SetItemString(
                publication.as_ptr(),
                b"category\0".as_ptr().cast(),
                category.as_ptr(),
            )
        };
        if category_status != 0 {
            return Err(PyErr::fetch(py));
        }
    }

    // PyLong_FromSize_t returns a new reference. SetItemString retains its own
    // reference on success; the original must be released on either status.
    let count_pointer = unsafe { ffi::PyLong_FromSize_t(records.len()) };
    if count_pointer.is_null() {
        return Err(PyErr::fetch(py));
    }
    let count_status = unsafe {
        ffi::PyDict_SetItemString(
            publication.as_ptr(),
            b"count\0".as_ptr().cast(),
            count_pointer,
        )
    };
    unsafe { ffi::Py_DECREF(count_pointer) };
    if count_status != 0 {
        return Err(PyErr::fetch(py));
    }

    for index in 0..records.len() {
        // GetItem borrows from records. The strong Bound acquired here survives
        // any Python reentry; Append retains a reference without stealing ours.
        let record_pointer = unsafe { ffi::PyList_GetItem(records.as_ptr(), index as isize) };
        let record = unsafe { Bound::from_borrowed_ptr_or_err(py, record_pointer) }?;
        let append_status = unsafe { ffi::PyList_Append(published_records.as_ptr(), record.as_ptr()) };
        if append_status != 0 {
            return Err(PyErr::fetch(py));
        }
    }

    // SetItemString retains the list without consuming the Bound reference.
    let records_status = unsafe {
        ffi::PyDict_SetItemString(
            publication.as_ptr(),
            b"records\0".as_ptr().cast(),
            published_records.as_ptr(),
        )
    };
    if records_status != 0 {
        return Err(PyErr::fetch(py));
    }
    Ok(publication)
}

pub fn describe_records(title: &str, records: &Bound<'_, PyList>) -> PublicationSummary {
    PublicationSummary {
        title: title.to_owned(),
        record_count: records.len(),
    }
}

pub fn render_summary(summary: &PublicationSummary) -> String {
    format!("{} ({} records)", summary.title, summary.record_count)
}
