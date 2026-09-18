use notify_debouncer_full::notify::Event;
use notify_debouncer_full::notify::event::{EventKind, ModifyKind};
use std::{
    collections::HashMap,
    path::PathBuf,
    sync::{Arc, Mutex},
};

/// A key for identifying blocked events, consisting of a normalized EventKind and a path.
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
struct BlockKey {
    kind: NormalizedEventKind,
    path: PathBuf,
}

/// A normalized representation of EventKind for use as a HashMap key.
/// This enum provides granular distinction for Modify::Name events with different RenameMode,
/// while normalizing other event types to their first level.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
enum NormalizedEventKind {
    Any,
    Access,
    Create,
    /// Modify events other than Name renames
    Modify,
    /// Modify::Name with RenameMode::From (file/folder that was renamed)
    ModifyNameFrom,
    /// Modify::Name with RenameMode::Both (single event with both paths)
    Remove,
    Other,
}

impl From<&EventKind> for NormalizedEventKind {
    fn from(kind: &EventKind) -> Self {
        match kind {
            EventKind::Any => NormalizedEventKind::Any,
            EventKind::Access(_) => NormalizedEventKind::Access,
            EventKind::Create(_) => NormalizedEventKind::Create,
            EventKind::Modify(modify_kind) => match modify_kind {
                ModifyKind::Name(_) => NormalizedEventKind::ModifyNameFrom,
                _ => NormalizedEventKind::Modify,
            },
            EventKind::Remove(_) => NormalizedEventKind::Remove,
            EventKind::Other => NormalizedEventKind::Other,
        }
    }
}

/// EventBlocker is used to filter out filesystem events that have already been
/// processed through other means (e.g., rename operations).
///
/// When a rename operation is processed, it may trigger additional filesystem events
/// (like Remove for the source and Create for the target). These events should be
/// blocked to avoid duplicate processing.
#[derive(Debug, Clone, Default)]
pub struct EventBlocker {
    /// Map of blocked event keys to their remaining block count.
    /// When count reaches 0, the entry is removed.
    blocked: Arc<Mutex<HashMap<BlockKey, usize>>>,
}

impl EventBlocker {
    /// Creates a new EventBlocker instance.
    pub fn new() -> Self {
        Self {
            blocked: Arc::new(Mutex::new(HashMap::new())),
        }
    }

    /// Registers an event stub to be blocked.
    ///
    /// The event will be blocked `count` times before being allowed through.
    /// If an entry already exists for this kind/path combination, the count is added.
    ///
    /// # Arguments
    /// * `kind` - The EventKind to block
    /// * `path` - The file path to block
    /// * `count` - Number of times to block this event (defaults to 1 if not specified)
    pub fn register(&self, kind: &EventKind, path: PathBuf, count: usize) {
        let key = BlockKey {
            kind: NormalizedEventKind::from(kind),
            path,
        };

        let mut blocked = self.blocked.lock().unwrap();
        *blocked.entry(key).or_insert(0) += count;
    }

    /// Convenience method to register an event to be blocked once.
    pub fn register_once(&self, kind: &EventKind, path: PathBuf) {
        self.register(kind, path, 1);
    }

    /// Checks if an event should be blocked and decrements the counter if so.
    ///
    /// # Arguments
    /// * `kind` - The EventKind of the event
    /// * `path` - The file path of the event
    ///
    /// # Returns
    /// `true` if the event should be blocked (was pre-registered), `false` otherwise
    pub fn should_block(&self, kind: &EventKind, path: &PathBuf) -> bool {
        let key = BlockKey {
            kind: NormalizedEventKind::from(kind),
            path: path.clone(),
        };

        let mut blocked = self.blocked.lock().unwrap();

        if let Some(count) = blocked.get_mut(&key) {
            if *count > 0 {
                *count -= 1;
                if *count == 0 {
                    blocked.remove(&key);
                }
                tracing::debug!(
                    target: "drive::event_blocker",
                    kind = ?kind,
                    path = %path.display(),
                    "Blocked pre-registered event"
                );
                return true;
            }
        }

        false
    }

    /// Filters a vector of events, removing those that have been pre-registered.
    ///
    /// For events with multiple paths, the event is only blocked if ALL paths are blocked.
    ///
    /// # Arguments
    /// * `events` - Vector of events to filter
    /// * `kind` - The EventKind for all events in this batch
    ///
    /// # Returns
    /// Filtered vector with blocked events removed
    pub fn filter_events(&self, events: Vec<Event>, kind: &EventKind) -> Vec<Event> {
        events
            .into_iter()
            .filter(|event| {
                // For events with paths, check if any path should be blocked
                for path in &event.paths {
                    if self.should_block(kind, path) {
                        // Event has at least one blocked path, filter it out
                        return false;
                    }
                }
                true
            })
            .collect()
    }

    /// Clears all registered event blocks.
    pub fn clear(&self) {
        let mut blocked = self.blocked.lock().unwrap();
        blocked.clear();
    }

    /// Returns the number of currently registered event blocks.
    pub fn len(&self) -> usize {
        let blocked = self.blocked.lock().unwrap();
        blocked.len()
    }

    /// Returns true if there are no registered event blocks.
    pub fn is_empty(&self) -> bool {
        self.len() == 0
    }
}
