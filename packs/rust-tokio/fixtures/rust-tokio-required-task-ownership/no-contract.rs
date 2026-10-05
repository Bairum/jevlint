pub async fn dispatch(
    counter: std::sync::Arc<std::sync::atomic::AtomicUsize>,
) -> std::io::Result<()> {
    let handle = tokio::task::spawn(async move {
        counter
            .try_update(
                std::sync::atomic::Ordering::Relaxed,
                std::sync::atomic::Ordering::Relaxed,
                |value| value.checked_add(1),
            )
            .map(|_| ())
            .map_err(|_| {
                std::io::Error::new(std::io::ErrorKind::Other, "counter exhausted")
            })
    });
    std::mem::drop(handle);
    Ok(())
}
