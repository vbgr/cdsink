//! Channel utilities for efficient batch processing.

use tokio::sync::mpsc;

/// Asynchronously waits for at least one message, then non-blockingly drains
/// the channel up to the specified limit.
///
/// This is the preferred method for consumers that want to balance latency and
/// throughput. It "sleeps" until data is available, then aggregates remaining
/// buffered messages to minimize downstream I/O calls.
///
/// ### Returns
/// * `Ok(Vec<T>)` - A batch containing 1 to `limit` items.
/// * `Err(TryRecvError::Disconnected)` - If the channel is closed and empty.
pub async fn recv_batch<T>(
    rx: &mut mpsc::Receiver<T>,
    limit: usize,
) -> Result<Vec<T>, mpsc::error::TryRecvError> {
    let mut batch = Vec::with_capacity(limit);

    // Block until the first message is available
    match rx.recv().await {
        Some(msg) => {
            batch.push(msg);

            // Non-blockingly drain the rest of the available buffer
            while batch.len() < limit {
                match rx.try_recv() {
                    Ok(next) => batch.push(next),
                    // If empty, we stop draining and return what we have
                    Err(mpsc::error::TryRecvError::Empty) => break,
                    // If disconnected, we return the batch we've collected so far
                    Err(mpsc::error::TryRecvError::Disconnected) => break,
                }
            }
            Ok(batch)
        }
        None => Err(mpsc::error::TryRecvError::Disconnected),
    }
}

/// Non-blockingly attempts to collect a batch of messages from the channel.
///
/// This function never suspends execution. It is useful for integration in
/// existing select! loops or poll-based logic where blocking is not permitted.
///
/// ### Returns
/// * `Ok(Vec<T>)` - A batch of available items (may be empty).
/// * `Err(TryRecvError::Disconnected)` - If the channel is closed.
pub fn try_recv_batch<T>(
    rx: &mut mpsc::Receiver<T>,
    limit: usize,
) -> Result<Vec<T>, mpsc::error::TryRecvError> {
    let mut batch = Vec::with_capacity(limit);

    while batch.len() < limit {
        match rx.try_recv() {
            Ok(msg) => batch.push(msg),
            Err(mpsc::error::TryRecvError::Empty) => break,
            Err(mpsc::error::TryRecvError::Disconnected) => {
                // If we collected items before disconnect, return them.
                // Otherwise, report the disconnection.
                if batch.is_empty() {
                    return Err(mpsc::error::TryRecvError::Disconnected);
                } else {
                    break;
                }
            }
        }
    }

    Ok(batch)
}
