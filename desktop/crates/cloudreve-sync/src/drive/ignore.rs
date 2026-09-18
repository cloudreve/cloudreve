//! Gitignore-style pattern matching for sync ignore rules.
//!
//! This module provides an `IgnoreMatcher` that can match file paths against
//! gitignore-style patterns. Patterns are relative to the sync root path,
//! and input paths are expected to be absolute paths.

use anyhow::{Context, Result};
use globset::{Glob, GlobSet, GlobSetBuilder};
use std::path::{Path, PathBuf};

/// A wrapper around `GlobSet` for matching ignore patterns (gitignore-style).
///
/// The matcher stores the sync root path and automatically strips it from
/// absolute paths before matching against the patterns.
#[derive(Debug, Clone)]
pub struct IgnoreMatcher {
    globset: GlobSet,
    /// Original patterns for debugging/logging
    patterns: Vec<String>,
    /// The sync root path - patterns are relative to this path
    sync_root: PathBuf,
}

impl IgnoreMatcher {
    /// Build an IgnoreMatcher from a list of gitignore-style patterns.
    ///
    /// # Arguments
    /// * `patterns` - List of gitignore-style patterns
    /// * `sync_root` - The sync root path. All patterns are relative to this path,
    ///                 and input paths will have this prefix stripped before matching.
    ///
    /// # Pattern Syntax
    /// - `*.log` - Matches any file ending with `.log` anywhere in the tree
    /// - `temp/` - Matches any directory named `temp` anywhere
    /// - `/build` - Matches `build` only at the sync root level
    /// - `docs/*.md` - Matches `.md` files in any `docs` directory
    /// - `#comment` - Lines starting with `#` are treated as comments
    pub fn new(patterns: &[String], sync_root: PathBuf) -> Result<Self> {
        let mut builder = GlobSetBuilder::new();

        for pattern in patterns {
            let pattern = pattern.trim();
            if pattern.is_empty() || pattern.starts_with('#') {
                // Skip empty lines and comments (gitignore-style)
                continue;
            }

            // Handle gitignore-style patterns:
            // - Patterns without '/' match anywhere in the path
            // - Patterns starting with '/' are anchored to root
            // - Patterns ending with '/' match directories only (we treat as prefix match)
            let glob_pattern = if pattern.contains('/') || pattern.contains('\\') {
                // Normalize path separators to forward slashes for glob matching
                let normalized = pattern.replace('\\', "/");

                // Pattern contains path separator
                if normalized.starts_with('/') {
                    // Anchored pattern - remove leading '/' and match from start
                    normalized[1..].to_string()
                } else {
                    // Match anywhere in the path
                    format!("**/{}", normalized)
                }
            } else {
                // Simple filename pattern - match anywhere
                format!("**/{}", pattern)
            };

            let glob = Glob::new(&glob_pattern)
                .with_context(|| format!("Invalid ignore pattern: {}", pattern))?;
            builder.add(glob);
        }

        // Add default set for office temp files
        builder.add(Glob::new("**/~*")?);
        builder.add(Glob::new("**/.~lock.*")?);
        builder.add(Glob::new("**/~*.tmp")?);

        let globset = builder
            .build()
            .context("Failed to build ignore pattern matcher")?;

        Ok(Self {
            globset,
            patterns: patterns.to_vec(),
            sync_root,
        })
    }

    /// Create an empty matcher that matches nothing.
    ///
    /// # Arguments
    /// * `sync_root` - The sync root path (still required for consistency)
    pub fn empty(sync_root: PathBuf) -> Self {
        Self {
            globset: GlobSet::empty(),
            patterns: Vec::new(),
            sync_root,
        }
    }

    /// Check if an absolute path matches any of the ignore patterns.
    ///
    /// The path will have the sync root prefix stripped before matching.
    /// If the path is not under the sync root, it will not match any patterns.
    ///
    /// # Arguments
    /// * `path` - The absolute path to check
    ///
    /// # Returns
    /// `true` if the path matches any ignore pattern, `false` otherwise
    pub fn is_match<P: AsRef<Path>>(&self, path: P) -> bool {
        let path = path.as_ref();

        // Try to get the relative path from sync root
        let relative_path = match path.strip_prefix(&self.sync_root) {
            Ok(rel) => rel,
            Err(_) => {
                // Path is not under sync root, cannot match
                return false;
            }
        };

        // Convert to forward slashes for consistent matching across platforms
        let normalized = relative_path.to_string_lossy().replace('\\', "/");

        self.globset.is_match(&normalized)
    }

    /// Check if a path (given as relative path from sync root) matches any patterns.
    ///
    /// Use this when you already have a relative path.
    ///
    /// # Arguments
    /// * `relative_path` - Path relative to sync root
    ///
    /// # Returns
    /// `true` if the path matches any ignore pattern, `false` otherwise
    pub fn is_match_relative<P: AsRef<Path>>(&self, relative_path: P) -> bool {
        let normalized = relative_path.as_ref().to_string_lossy().replace('\\', "/");

        self.globset.is_match(&normalized)
    }

    /// Check if a filename (without path) matches any of the ignore patterns.
    ///
    /// This is useful for quick checks on just the filename.
    /// Note: This only matches patterns that don't contain path separators.
    ///
    /// # Arguments
    /// * `filename` - The filename to check (without path)
    ///
    /// # Returns
    /// `true` if the filename matches any ignore pattern, `false` otherwise
    pub fn is_match_filename(&self, filename: &str) -> bool {
        self.globset.is_match(filename)
    }

    /// Get the original patterns for debugging/logging.
    pub fn patterns(&self) -> &[String] {
        &self.patterns
    }

    /// Get the sync root path.
    pub fn sync_root(&self) -> &Path {
        &self.sync_root
    }

    /// Check if the matcher has any patterns.
    pub fn is_empty(&self) -> bool {
        self.globset.is_empty()
    }

    /// Get the number of patterns.
    pub fn len(&self) -> usize {
        self.globset.len()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_simple_pattern() {
        let sync_root = PathBuf::from("C:\\Users\\test\\sync");
        let patterns = vec!["*.log".to_string()];
        let matcher = IgnoreMatcher::new(&patterns, sync_root.clone()).unwrap();

        assert!(matcher.is_match("C:\\Users\\test\\sync\\debug.log"));
        assert!(matcher.is_match("C:\\Users\\test\\sync\\subdir\\error.log"));
        assert!(!matcher.is_match("C:\\Users\\test\\sync\\readme.txt"));
    }

    #[test]
    fn test_anchored_pattern() {
        let sync_root = PathBuf::from("C:\\Users\\test\\sync");
        let patterns = vec!["/build".to_string()];
        let matcher = IgnoreMatcher::new(&patterns, sync_root.clone()).unwrap();

        assert!(matcher.is_match("C:\\Users\\test\\sync\\build"));
        assert!(!matcher.is_match("C:\\Users\\test\\sync\\src\\build"));
    }

    #[test]
    fn test_directory_pattern() {
        let sync_root = PathBuf::from("C:\\Users\\test\\sync");
        let patterns = vec!["node_modules".to_string()];
        let matcher = IgnoreMatcher::new(&patterns, sync_root.clone()).unwrap();

        assert!(matcher.is_match("C:\\Users\\test\\sync\\node_modules"));
        assert!(matcher.is_match("C:\\Users\\test\\sync\\project\\node_modules"));
    }

    #[test]
    fn test_path_pattern() {
        let sync_root = PathBuf::from("C:\\Users\\test\\sync");
        let patterns = vec!["docs/*.md".to_string()];
        let matcher = IgnoreMatcher::new(&patterns, sync_root.clone()).unwrap();

        assert!(matcher.is_match("C:\\Users\\test\\sync\\docs\\readme.md"));
        assert!(matcher.is_match("C:\\Users\\test\\sync\\project\\docs\\api.md"));
        assert!(!matcher.is_match("C:\\Users\\test\\sync\\readme.md"));
    }

    #[test]
    fn test_comment_and_empty_lines() {
        let sync_root = PathBuf::from("C:\\Users\\test\\sync");
        let patterns = vec![
            "# This is a comment".to_string(),
            "".to_string(),
            "  ".to_string(),
            "*.tmp".to_string(),
        ];
        let matcher = IgnoreMatcher::new(&patterns, sync_root.clone()).unwrap();

        assert_eq!(matcher.len(), 1); // Only *.tmp should be added
        assert!(matcher.is_match("C:\\Users\\test\\sync\\file.tmp"));
    }

    #[test]
    fn test_path_outside_sync_root() {
        let sync_root = PathBuf::from("C:\\Users\\test\\sync");
        let patterns = vec!["*.log".to_string()];
        let matcher = IgnoreMatcher::new(&patterns, sync_root.clone()).unwrap();

        // Path outside sync root should never match
        assert!(!matcher.is_match("C:\\Other\\path\\debug.log"));
    }

    #[test]
    fn test_relative_path_matching() {
        let sync_root = PathBuf::from("C:\\Users\\test\\sync");
        let patterns = vec!["*.log".to_string(), "/build".to_string()];
        let matcher = IgnoreMatcher::new(&patterns, sync_root).unwrap();

        assert!(matcher.is_match_relative("debug.log"));
        assert!(matcher.is_match_relative("subdir/error.log"));
        assert!(matcher.is_match_relative("build"));
        assert!(!matcher.is_match_relative("src/build"));
    }
}
