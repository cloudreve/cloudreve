use notify_debouncer_full::notify::Event;
use notify_debouncer_full::notify::event::{EventKind, ModifyKind};
use std::{
    collections::HashMap,
    path::PathBuf,
    sync::{Arc, Mutex},
    time::{Duration, Instant},
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
///
/// A path-prefix suppression entry: blocks every event of `kind` whose path
/// lies under `prefix` until `expires_at`. Used to swallow the per-descendant
/// Remove/Create events the OS emits when a directory is moved or renamed.
#[derive(Debug)]
struct PrefixBlock {
    kind: NormalizedEventKind,
    prefix: PathBuf,
    expires_at: Instant,
}

#[derive(Debug, Clone, Default)]
pub struct EventBlocker {
    /// Map of blocked event keys to their remaining block count.
    /// When count reaches 0, the entry is removed.
    blocked: Arc<Mutex<HashMap<BlockKey, usize>>>,
    /// Path-prefix suppressions with a fixed expiry.
    prefix_blocked: Arc<Mutex<Vec<PrefixBlock>>>,
}

impl EventBlocker {
    /// Creates a new EventBlocker instance.
    pub fn new() -> Self {
        Self {
            blocked: Arc::new(Mutex::new(HashMap::new())),
            prefix_blocked: Arc::new(Mutex::new(Vec::new())),
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

    /// Registers a path-prefix suppression for `ttl`.
    ///
    /// While the entry is alive, every event of `kind` whose path is `prefix`
    /// itself or a descendant of it is blocked. Unlike [register], entries are
    /// not consumed per event — they live until expiry, since the number of
    /// descendant events the OS emits for a moved directory is not knowable.
    ///
    /// Keep the TTL tight: a genuine operation inside the tree during the
    /// window is swallowed as well.
    pub fn register_prefix(&self, kind: &EventKind, prefix: PathBuf, ttl: Duration) {
        let mut blocked = self.prefix_blocked.lock().unwrap();
        blocked.retain(|b| b.expires_at > Instant::now());
        blocked.push(PrefixBlock {
            kind: NormalizedEventKind::from(kind),
            prefix,
            expires_at: Instant::now() + ttl,
        });
    }

    /// Returns true if `path` is covered by a live prefix suppression of `kind`.
    fn is_prefix_blocked(&self, kind: &EventKind, path: &PathBuf) -> bool {
        let normalized = NormalizedEventKind::from(kind);
        let mut blocked = self.prefix_blocked.lock().unwrap();
        blocked.retain(|b| b.expires_at > Instant::now());
        blocked
            .iter()
            .any(|b| b.kind == normalized && path.starts_with(&b.prefix))
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
        if self.is_prefix_blocked(kind, path) {
            tracing::debug!(
                target: "drive::event_blocker",
                kind = ?kind,
                path = %path.display(),
                "Blocked event under suppressed prefix"
            );
            return true;
        }

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
        let mut prefix_blocked = self.prefix_blocked.lock().unwrap();
        prefix_blocked.clear();
    }

    /// Returns the number of currently registered event blocks.
    pub fn len(&self) -> usize {
        let blocked = self.blocked.lock().unwrap();
        let prefix_blocked = self.prefix_blocked.lock().unwrap();
        blocked.len() + prefix_blocked.len()
    }

    /// Returns true if there are no registered event blocks.
    pub fn is_empty(&self) -> bool {
        self.len() == 0
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use notify_debouncer_full::notify::event::RemoveKind;

    #[test]
    fn prefix_blocks_descendant_and_self() {
        let blocker = EventBlocker::new();
        let dir = PathBuf::from("/sync/dir");
        blocker.register_prefix(
            &EventKind::Remove(RemoveKind::Any),
            dir.clone(),
            Duration::from_secs(60),
        );

        assert!(blocker.should_block(
            &EventKind::Remove(RemoveKind::Any),
            &dir.join("child/file.txt")
        ));
        assert!(blocker.should_block(&EventKind::Remove(RemoveKind::Any), &dir));
        // Different kind and different subtree are not blocked
        assert!(!blocker.should_block(
            &EventKind::Create(notify_debouncer_full::notify::event::CreateKind::Any),
            &dir.join("child/file.txt")
        ));
        assert!(!blocker.should_block(
            &EventKind::Remove(RemoveKind::Any),
            &PathBuf::from("/sync/other/file.txt")
        ));
        // Sibling sharing a name prefix must not match ("dir2" vs "dir")
        assert!(!blocker.should_block(
            &EventKind::Remove(RemoveKind::Any),
            &PathBuf::from("/sync/dir2/file.txt")
        ));
    }

    #[test]
    fn prefix_expires() {
        let blocker = EventBlocker::new();
        let dir = PathBuf::from("/sync/dir");
        blocker.register_prefix(
            &EventKind::Remove(RemoveKind::Any),
            dir.clone(),
            Duration::from_millis(0),
        );
        assert!(!blocker.should_block(
            &EventKind::Remove(RemoveKind::Any),
            &dir.join("child")
        ));
    }

    #[test]
    fn exact_block_consumed_once() {
        let blocker = EventBlocker::new();
        let path = PathBuf::from("/sync/file.txt");
        blocker.register_once(&EventKind::Remove(RemoveKind::Any), path.clone());
        assert!(blocker.should_block(&EventKind::Remove(RemoveKind::Any), &path));
        assert!(!blocker.should_block(&EventKind::Remove(RemoveKind::Any), &path));
    }
}
