pub struct CloseHandle {
    pub sender: Option<std::sync::mpsc::Sender<()>>,
}

impl CloseHandle {
    /// Request graceful shutdown explicitly and report a disconnected server.
    pub fn close(&mut self) -> Result<(), std::sync::mpsc::SendError<()>> {
        match self.sender.take() {
            Some(sender) => sender.send(()),
            None => Ok(()),
        }
    }
}

impl std::ops::Drop for CloseHandle {
    /// Request shutdown on release. The server receiver may already be gone
    /// during runtime teardown; this best-effort fallback must not panic.
    fn drop(&mut self) {
        if let Some(sender) = self.sender.take() {
            let _ = sender.send(());
        }
    }
}

pub fn server_handles() -> (CloseHandle, std::sync::mpsc::Receiver<()>) {
    let (sender, receiver) = std::sync::mpsc::channel();
    (CloseHandle { sender: Some(sender) }, receiver)
}

pub fn release_server_receiver(
    handle: CloseHandle,
    receiver: std::sync::mpsc::Receiver<()>,
) -> CloseHandle {
    std::mem::drop(receiver);
    handle
}
