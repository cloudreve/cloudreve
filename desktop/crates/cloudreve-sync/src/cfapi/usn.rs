// TODO: this module will be for handling usn's and potentially change journals.
// Usn's should ALWAYS be required and this module should make it easy

/// An Updated Sequence Number (USN) is as an identifier that represents the version of a file. Each
/// subsequent file operation will increment the USN, allowing you to recognize when a file has
/// been updated.
///
/// A USN is commonly used to prevent a change from happening unless the USN is up to date. For
/// instance, [Placeholder::update][crate::placeholder::Placeholder::update] will not apply the specified changes
/// unless if the passed USN matches the most recent USN of the file. This avoids applying changes
/// that may be out of date.
pub type Usn = i64;
