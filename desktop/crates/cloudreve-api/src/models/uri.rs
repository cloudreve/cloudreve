use anyhow::Result;
use percent_encoding::{percent_decode_str, percent_encode, AsciiSet, NON_ALPHANUMERIC};
use std::{
    collections::HashMap,
    fmt::{Display, Formatter},
};
use url::Url;

use super::explorer::file_type;

/// Character set for encoding that matches JavaScript's encodeURIComponent.
/// encodeURIComponent escapes all characters except: A-Z a-z 0-9 - _ . ! ~ * ' ( )
const ENCODE_URI_COMPONENT_SET: &AsciiSet = &NON_ALPHANUMERIC
    .remove(b'-')
    .remove(b'_')
    .remove(b'.')
    .remove(b'!')
    .remove(b'~')
    .remove(b'*')
    .remove(b'\'')
    .remove(b'(')
    .remove(b')');

/// Encode a string using the same rules as JavaScript's encodeURIComponent
fn encode_uri_component(s: &str) -> String {
    percent_encode(s.as_bytes(), ENCODE_URI_COMPONENT_SET).to_string()
}

/// Decode a percent-encoded string
fn decode_uri_component(s: &str) -> String {
    percent_decode_str(s)
        .decode_utf8()
        .unwrap_or_else(|_| s.into())
        .to_string()
}

/// Cloudreve URI prefix
pub const CR_URI_PREFIX: &str = "cloudreve://";
const HTTP_URI_PREFIX: &str = "http://";

/// Filesystem types
pub mod filesystem {
    pub const MY: &str = "my";
    pub const SHARE: &str = "share";
    pub const SHARED_BY_ME: &str = "shared_by_me";
    pub const SHARED_WITH_ME: &str = "shared_with_me";
    pub const TRASH: &str = "trash";
}

/// URI Query parameters
pub mod uri_query {
    pub const NAME: &str = "name";
    pub const NAME_OP_OR: &str = "name_op_or";
    pub const METADATA_PREFIX: &str = "meta_";
    pub const METADATA_STRONG_MATCH: &str = "exact_meta_";
    pub const CASE_FOLDING: &str = "case_folding";
    pub const TYPE: &str = "type";
    pub const CATEGORY: &str = "category";
    pub const SIZE_GTE: &str = "size_gte";
    pub const SIZE_LTE: &str = "size_lte";
    pub const CREATED_GTE: &str = "created_gte";
    pub const CREATED_LTE: &str = "created_lte";
    pub const UPDATED_GTE: &str = "updated_gte";
    pub const UPDATED_LTE: &str = "updated_lte";
}

/// Search categories
pub mod uri_search_category {
    pub const IMAGE: &str = "image";
    pub const VIDEO: &str = "video";
    pub const AUDIO: &str = "audio";
    pub const DOCUMENT: &str = "document";
}

/// Search parameters
#[derive(Debug, Clone, Default)]
pub struct SearchParam {
    pub name: Option<Vec<String>>,
    pub name_op_or: Option<bool>,
    pub metadata: Option<HashMap<String, String>>,
    pub metadata_strong_match: Option<HashMap<String, String>>,
    pub case_folding: Option<bool>,
    pub category: Option<String>,
    pub type_: Option<i32>,
    pub size_gte: Option<u64>,
    pub size_lte: Option<u64>,
    pub created_at_gte: Option<i64>,
    pub created_at_lte: Option<i64>,
    pub updated_at_gte: Option<i64>,
    pub updated_at_lte: Option<i64>,
}

/// Custom error type for URI operations
#[derive(Debug, Clone)]
pub enum UriError {
    InvalidPrefix(String),
    ParseError(String),
}

impl Display for UriError {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        match self {
            UriError::InvalidPrefix(uri) => write!(f, "Invalid cloudreve URI prefix: {}", uri),
            UriError::ParseError(msg) => write!(f, "URI parse error: {}", msg),
        }
    }
}

impl std::error::Error for UriError {}

// Implement conversion from url::ParseError
impl From<url::ParseError> for UriError {
    fn from(err: url::ParseError) -> Self {
        UriError::ParseError(err.to_string())
    }
}

/// Cloudreve URI
#[derive(Debug, Clone)]
pub struct CrUri {
    url: Url,
}

impl CrUri {
    /// Create a new CrUri from a string
    pub fn new(u: &str) -> Result<Self> {
        if !u.starts_with(CR_URI_PREFIX) {
            return Err(anyhow::anyhow!("Invalid cloudreve URI prefix: {}", u));
        }

        // Replace prefix with standard HTTP for compatibility
        let u = u.replace(CR_URI_PREFIX, HTTP_URI_PREFIX);
        let mut url = Url::parse(&u)?;

        // Remove ending slash if present
        let path = url.path().trim_end_matches('/').to_string();
        url.set_path(&path);

        Ok(Self { url })
    }

    /// Get the ID (username) from the URI
    pub fn id(&self) -> String {
        self.url.username().to_string()
    }

    /// Get the password from the URI
    pub fn password(&self) -> String {
        self.url.password().unwrap_or("").to_string()
    }

    /// Check if the URI has search parameters
    pub fn is_search(&self) -> bool {
        self.url.query_pairs().next().is_some()
    }

    /// Get all values for a query parameter
    pub fn query(&self, key: &str) -> Vec<String> {
        self.url
            .query_pairs()
            .filter(|(k, _)| k == key)
            .map(|(_, v)| v.to_string())
            .collect()
    }

    /// Add a query parameter
    pub fn add_query(&mut self, key: &str, value: &str) -> &mut Self {
        {
            self.url.query_pairs_mut().append_pair(key, value);
        }
        self
    }

    /// Set search parameters
    pub fn set_search_param(&mut self, param: SearchParam) -> &mut Self {
        // Clear all existing query parameters
        self.url.set_query(None);

        if let Some(names) = param.name {
            if self.fs() == filesystem::TRASH {
                // In trash, search by restore_uri metadata
                let encoded = urlencoding::encode(&names.join(" ")).to_string();
                self.add_query(
                    &format!("{}restore_uri", uri_query::METADATA_PREFIX),
                    &encoded,
                );
            } else {
                for name in names {
                    self.add_query(uri_query::NAME, &name);
                }
            }
        }

        if param.name_op_or.unwrap_or(false) {
            self.add_query(uri_query::NAME_OP_OR, "");
        }

        if param.case_folding.unwrap_or(false) {
            self.add_query(uri_query::CASE_FOLDING, "");
        }

        if let Some(category) = param.category {
            self.add_query(uri_query::CATEGORY, &category);
        }

        if let Some(type_) = param.type_ {
            let type_str = if type_ == file_type::FOLDER {
                "folder"
            } else {
                "file"
            };
            self.add_query(uri_query::TYPE, type_str);
        }

        if let Some(metadata) = param.metadata {
            for (k, v) in metadata {
                self.add_query(&format!("{}{}", uri_query::METADATA_PREFIX, k), &v);
            }
        }

        if let Some(metadata_strong_match) = param.metadata_strong_match {
            for (k, v) in metadata_strong_match {
                self.add_query(&format!("{}{}", uri_query::METADATA_STRONG_MATCH, k), &v);
            }
        }

        if let Some(size_gte) = param.size_gte {
            self.add_query(uri_query::SIZE_GTE, &size_gte.to_string());
        }

        if let Some(size_lte) = param.size_lte {
            self.add_query(uri_query::SIZE_LTE, &size_lte.to_string());
        }

        if let Some(created_at_gte) = param.created_at_gte {
            self.add_query(uri_query::CREATED_GTE, &created_at_gte.to_string());
        }

        if let Some(created_at_lte) = param.created_at_lte {
            self.add_query(uri_query::CREATED_LTE, &created_at_lte.to_string());
        }

        if let Some(updated_at_gte) = param.updated_at_gte {
            self.add_query(uri_query::UPDATED_GTE, &updated_at_gte.to_string());
        }

        if let Some(updated_at_lte) = param.updated_at_lte {
            self.add_query(uri_query::UPDATED_LTE, &updated_at_lte.to_string());
        }

        self
    }

    /// Get search parameters from the URI
    pub fn search_params(&self) -> Option<SearchParam> {
        if !self.is_search() {
            return None;
        }

        let mut res = SearchParam::default();

        for (k, v) in self.url.query_pairs() {
            match k.as_ref() {
                uri_query::NAME => {
                    if res.name.is_none() {
                        res.name = Some(Vec::new());
                    }
                    res.name.as_mut().unwrap().push(v.to_string());
                }
                uri_query::NAME_OP_OR => {
                    res.name_op_or = Some(true);
                }
                uri_query::CASE_FOLDING => {
                    res.case_folding = Some(true);
                }
                uri_query::CATEGORY => {
                    res.category = Some(v.to_string());
                }
                uri_query::TYPE => {
                    res.type_ = Some(if v == "file" {
                        file_type::FILE
                    } else {
                        file_type::FOLDER
                    });
                }
                uri_query::SIZE_GTE => {
                    res.size_gte = v.parse().ok();
                }
                uri_query::SIZE_LTE => {
                    res.size_lte = v.parse().ok();
                }
                uri_query::CREATED_GTE => {
                    res.created_at_gte = v.parse().ok();
                }
                uri_query::CREATED_LTE => {
                    res.created_at_lte = v.parse().ok();
                }
                uri_query::UPDATED_GTE => {
                    res.updated_at_gte = v.parse().ok();
                }
                uri_query::UPDATED_LTE => {
                    res.updated_at_lte = v.parse().ok();
                }
                _ => {
                    if k.starts_with(uri_query::METADATA_PREFIX) {
                        if res.metadata.is_none() {
                            res.metadata = Some(HashMap::new());
                        }
                        let key = k[uri_query::METADATA_PREFIX.len()..].to_string();
                        res.metadata.as_mut().unwrap().insert(key, v.to_string());
                    } else if k.starts_with(uri_query::METADATA_STRONG_MATCH) {
                        if res.metadata_strong_match.is_none() {
                            res.metadata_strong_match = Some(HashMap::new());
                        }
                        let key = k[uri_query::METADATA_STRONG_MATCH.len()..].to_string();
                        res.metadata_strong_match
                            .as_mut()
                            .unwrap()
                            .insert(key, v.to_string());
                    }
                }
            }
        }

        Some(res)
    }

    /// Get the path from the URI (decoded)
    pub fn path(&self) -> String {
        decode_uri_component(self.url.path())
    }

    /// Set the path of the URI
    pub fn set_path(&mut self, path: &str) -> &mut Self {
        let encoded_segments: Vec<String> = path
            .split('/')
            .map(|p| encode_uri_component(p))
            .collect();
        let encoded_path = encoded_segments.join("/");
        self.url.set_path(&encoded_path);
        self
    }

    /// Set the username
    pub fn set_username(&mut self, username: &str) -> Result<&mut Self, ()> {
        self.url.set_username(username)?;
        Ok(self)
    }

    /// Set the password
    pub fn set_password(&mut self, password: &str) -> Result<&mut Self, ()> {
        self.url.set_password(Some(password))?;
        Ok(self)
    }

    /// Get the path without leading slash
    pub fn path_trimmed(&self) -> String {
        self.url.path().trim_start_matches('/').to_string()
    }

    /// Join paths to the current path
    pub fn join(&mut self, paths: &[&str]) -> &mut Self {
        let current_path = self.url.path();
        let mut result = current_path.to_string();

        for p in paths {
            let encoded = encode_uri_component(p);
            if !result.ends_with('/') {
                result.push('/');
            }
            result.push_str(&encoded);
        }

        self.url.set_path(&result);
        self
    }

    /// Join a raw path (may be absolute or relative)
    pub fn join_raw(&mut self, raw_path: &str) -> &mut Self {
        if raw_path.starts_with('/') {
            // Absolute path - replace the entire pathname
            self.url.set_path(raw_path);
        } else {
            // Relative path - resolve it against the current path
            let current = self.url.path().trim_end_matches('/');
            let new_path = if current.is_empty() {
                format!("/{}", raw_path)
            } else {
                format!("{}/{}", current, raw_path)
            };
            self.url.set_path(&new_path);
        }
        self
    }

    /// Get path elements as a vector of strings
    pub fn elements(&self) -> Vec<String> {
        let trimmed = self.path_trimmed();
        if trimmed.is_empty() {
            return Vec::new();
        }

        trimmed
            .split('/')
            .map(|p| decode_uri_component(p))
            .collect()
    }

    /// Check if the URI is pointing to root
    pub fn is_root(&self) -> bool {
        let path = self.url.path();
        path.is_empty() || path == "/"
    }

    /// Get the filesystem type (hostname)
    pub fn fs(&self) -> String {
        self.url.host_str().unwrap_or("").to_string()
    }

    /// Get the root ID
    pub fn root_id(&self) -> String {
        // TODO: Implement SessionManager integration
        let user_id = "0"; // SessionManager.currentLoginOrNull()?.user.id ?? "0"
        format!("{}/{}/{}", self.fs(), self.url.username(), user_id)
    }

    /// Get the base URI (without path)
    pub fn base(&self, exclude_search: bool) -> String {
        let mut new_url = self.url.clone();
        new_url.set_path("");
        if exclude_search {
            new_url.set_query(None);
        }

        new_url
            .to_string()
            .replace(HTTP_URI_PREFIX, CR_URI_PREFIX)
            .trim_end_matches('/')
            .to_string()
    }

    /// Returns the URI without searching query string, with exceptions
    pub fn pure_uri(&self, exceptions: &[&str]) -> Result<CrUri> {
        let mut new_uri = CrUri::new(&self.to_string())?;

        let keys_for_del: Vec<String> = new_uri
            .url
            .query_pairs()
            .filter(|(k, _)| !exceptions.contains(&k.as_ref()))
            .map(|(k, _)| k.to_string())
            .collect();

        for key in keys_for_del {
            // Create a new query string without the deleted keys
            let filtered_pairs: Vec<(String, String)> = new_uri
                .url
                .query_pairs()
                .filter(|(k, _)| k != &key)
                .map(|(k, v)| (k.to_string(), v.to_string()))
                .collect();

            new_uri.url.set_query(None);
            for (k, v) in filtered_pairs {
                new_uri.add_query(&k, &v);
            }
        }

        Ok(new_uri)
    }

    /// Get the parent URI
    pub fn parent(&self) -> Result<CrUri> {
        let mut new_uri = CrUri::new(&self.to_string())?;
        let mut path = new_uri.elements();
        path.pop();

        let new_path = if !path.is_empty() {
            format!("/{}", path.join("/"))
        } else {
            String::new()
        };

        new_uri.set_path(&new_path);
        Ok(new_uri)
    }

    /// Convert the URI to a string
    pub fn to_string(&self) -> String {
        self.url
            .to_string()
            .replace(HTTP_URI_PREFIX, CR_URI_PREFIX)
            .trim_end_matches('/')
            .to_string()
    }
}

/// Create a new "my" filesystem URI
pub fn new_my_uri(uid: Option<&str>) -> Result<CrUri> {
    match uid {
        Some(uid) => CrUri::new(&format!("cloudreve://{}@my", uid)),
        None => CrUri::new("cloudreve://my"),
    }
}
