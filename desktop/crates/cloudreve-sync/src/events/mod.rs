use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tokio::sync::broadcast;
use tracing;

/// Different types of events that can be broadcast to GUI
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "type", content = "data")]
pub enum Event {
    ConnectionStatusChanged {
        connected: bool,
    },
    NoDrive {
    },
    /// Request to open the sync status window
    OpenSyncStatusWindow,
    /// Request to open the settings window
    OpenSettingsWindow,
}

impl Event {
    pub fn name(&self) -> &'static str {
        match self {
            Event::ConnectionStatusChanged { .. } => "ConnectionStatusChanged",
            Event::NoDrive {  } => "NoDrive",
            Event::OpenSyncStatusWindow => "OpenSyncStatusWindow",
            Event::OpenSettingsWindow => "OpenSettingsWindow",
        }
    }
}

/// Event broadcaster for Server-Sent Events (SSE)
#[derive(Clone)]
pub struct EventBroadcaster {
    sender: Arc<broadcast::Sender<Event>>,
}

impl EventBroadcaster {
    /// Create a new event broadcaster
    ///
    /// # Arguments
    /// * `capacity` - The capacity of the broadcast channel (default: 100)
    pub fn new(capacity: usize) -> Self {
        let (sender, _) = broadcast::channel(capacity);
        Self {
            sender: Arc::new(sender),
        }
    }

    /// Subscribe to events and get a receiver
    pub fn subscribe(&self) -> broadcast::Receiver<Event> {
        self.sender.subscribe()
    }

    /// Broadcast an event to all subscribers
    ///
    /// # Arguments
    /// * `event` - The event to broadcast
    ///
    /// # Returns
    /// The number of receivers that received the event
    pub fn broadcast(&self, event: Event) -> usize {
        match self.sender.send(event.clone()) {
            Ok(count) => {
                tracing::debug!(target: "events", subscribers = count, "Broadcast event to subscriber(s)");
                tracing::trace!(target: "events", event = ?event, "Event details");
                count
            }
            Err(e) => {
                tracing::warn!(target: "events", error = ?e, "Failed to broadcast event (no active subscribers)");
                0
            }
        }
    }

    /// Helper: Broadcast no drive event
    pub fn no_drive(&self) {
        self.broadcast(Event::NoDrive {  });
    }

    /// Helper: Broadcast connection status changed event
    pub fn connection_status_changed(&self, connected: bool) {
        self.broadcast(Event::ConnectionStatusChanged { connected });
    }

    /// Helper: Broadcast open sync status window event
    pub fn open_sync_status_window(&self) {
        self.broadcast(Event::OpenSyncStatusWindow);
    }

    /// Helper: Broadcast open settings window event
    pub fn open_settings_window(&self) {
        self.broadcast(Event::OpenSettingsWindow);
    }

    /// Get the number of active subscribers
    pub fn subscriber_count(&self) -> usize {
        self.sender.receiver_count()
    }
}

impl Default for EventBroadcaster {
    fn default() -> Self {
        Self::new(100)
    }
}