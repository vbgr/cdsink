//! Channel utilities for efficient batch processing.

use tokio::sync::mpsc;
use tracing::instrument;

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
#[instrument(skip(rx))]
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

/// Sends a collection of items into a channel, consuming them.
///
/// Returns `Ok(())` if all items were sent.
/// Returns `Err(_)` if the receiver has dropped, indicating the pipeline
/// should initiate a graceful shutdown.
pub async fn send_all<T>(
    tx: &mpsc::Sender<T>,
    items: Vec<T>,
) -> Result<(), mpsc::error::SendError<T>> {
    for item in items {
        tx.send(item).await?;
    }
    Ok(())
}

/// Creates a fixed number of MPSC channels with a specified capacity.
///
/// Returns a tuple containing:
/// 1. A Vector of Senders (to be kept by the dispatcher/main loop).
/// 2. A Vector of Receivers (to be moved into worker tasks).
pub fn make_channels<T>(
    capacity: usize,
    len: usize,
) -> (Vec<mpsc::Sender<T>>, Vec<mpsc::Receiver<T>>) {
    let mut txs = Vec::with_capacity(len);
    let mut rxs = Vec::with_capacity(len);

    for _ in 0..len {
        let (tx, rx) = mpsc::channel::<T>(capacity);
        txs.push(tx);
        rxs.push(rx);
    }

    (txs, rxs)
}
