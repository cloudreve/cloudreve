use crate::utils::app::get_app_root;
use anyhow::{Context, Result};
use serde::Deserialize;
use sha2::{Digest, Sha256};
use std::path::PathBuf;

/// Manifest.json structure
#[derive(Debug, Deserialize)]
struct ManifestIcon {
    sizes: String,
    src: String,
    #[serde(rename = "type")]
    icon_type: String,
}

#[derive(Debug, Deserialize)]
struct Manifest {
    icons: Vec<ManifestIcon>,
}

/// Result containing paths to both the ICO icon and raw image
#[derive(Debug, Clone)]
pub struct FaviconResult {
    /// Path to the ICO file (for Windows shell integration)
    pub ico_path: String,
    /// Path to the raw image file (PNG/JPG/etc, before ICO conversion)
    pub raw_path: String,
}

/// Icon type for download and fallback handling
#[derive(Debug, Clone, Copy)]
enum IconType {
    /// Small icon for ICO conversion (Windows shell integration)
    Small,
    /// Large icon for raw display (status UI)
    Large,
}

/// Get the icons directory path
fn get_icons_dir() -> Result<PathBuf> {
    let home_dir = dirs::home_dir().context("Failed to get user home directory")?;
    let icons_dir = home_dir.join(".cloudreve").join("icos");

    // Ensure icons directory exists
    if !icons_dir.exists() {
        std::fs::create_dir_all(&icons_dir).context("Failed to create icons directory")?;
    }

    Ok(icons_dir)
}

/// Parse icon size from sizes string (e.g., "192x192" or "64x64 32x32")
/// Returns the first (typically largest for multi-size) dimension
fn parse_icon_size(sizes: &str) -> Option<u32> {
    sizes
        .split_whitespace()
        .filter_map(|size| size.split('x').next().and_then(|s| s.parse::<u32>().ok()))
        .next()
}

/// Build full URL from icon src and base instance URL
fn build_icon_url(icon: &ManifestIcon, instance_url: &str) -> String {
    if icon.src.starts_with("http") {
        icon.src.clone()
    } else {
        let base = instance_url.trim_end_matches('/');
        let path = icon.src.trim_start_matches('/');
        if icon.src.starts_with('/') {
            format!("{}{}", base, icon.src)
        } else {
            format!("{}/{}", base, path)
        }
    }
}

/// Determine file extension from icon type and URL
fn get_extension_from_icon(icon: &ManifestIcon, url: &str) -> &'static str {
    if icon.icon_type.contains("png") {
        "png"
    } else if icon.icon_type.contains("jpeg") || icon.icon_type.contains("jpg") {
        "jpg"
    } else if icon.icon_type.contains("x-icon") || icon.icon_type.contains("ico") {
        "ico"
    } else if url.ends_with(".png") {
        "png"
    } else if url.ends_with(".jpg") || url.ends_with(".jpeg") {
        "jpg"
    } else if url.ends_with(".ico") {
        "ico"
    } else {
        "png" // Default to PNG
    }
}

/// Check if the icon is already in ICO format
fn is_ico_format(icon: &ManifestIcon, url: &str) -> bool {
    icon.icon_type.contains("x-icon") || url.ends_with(".ico")
}

/// Download an icon from URL
async fn download_icon(client: &reqwest::Client, url: &str) -> Result<bytes::Bytes> {
    tracing::debug!(target: "drive::favicon", icon_url = %url, "Downloading icon");
    let response = client
        .get(url)
        .send()
        .await
        .context("Failed to download icon")?;

    let status = response.status();
    if !status.is_success() {
        anyhow::bail!("Failed to download icon: HTTP {}", status);
    }

    response
        .bytes()
        .await
        .context("Failed to read icon bytes")
}

/// Get fallback icon bytes for the given icon type
fn get_fallback_icon(icon_type: IconType) -> Result<Vec<u8>> {
    let app_root = get_app_root();
    let fallback_path = match icon_type {
        IconType::Small => format!("{}\\cloudreve.ico", app_root.image_path_general()),
        IconType::Large => format!("{}\\StoreLogo.scale-400.png", app_root.image_path_general()),
    };

    tracing::info!(target: "drive::favicon", path = %fallback_path, icon_type = ?icon_type, "Using fallback icon");
    std::fs::read(&fallback_path).with_context(|| format!("Failed to read fallback icon: {}", fallback_path))
}

/// Save icon bytes to destination, converting to ICO if needed
fn save_icon(bytes: &[u8], dest_path: &PathBuf, convert_to_ico: bool, is_already_ico: bool) -> Result<()> {
    if convert_to_ico && !is_already_ico {
        // Convert image to ICO format
        let img = image::load_from_memory(bytes).context("Failed to load image")?;
        // Resize to 64x64 for ICO
        let resized = img.resize(64, 64, image::imageops::FilterType::Lanczos3);
        resized
            .save_with_format(dest_path, image::ImageFormat::Ico)
            .context("Failed to save as ICO")?;
    } else {
        std::fs::write(dest_path, bytes).context("Failed to save icon file")?;
    }
    Ok(())
}

/// Fetch and save favicon from instance_url
/// Returns both the ICO path and the raw image path
/// For ICO: downloads the smallest icon for Windows shell integration
/// For raw: downloads the largest icon for status UI display
/// Falls back to bundled icons if download fails
pub async fn fetch_and_save_favicon(instance_url: &str) -> Result<FaviconResult> {
    tracing::info!(target: "drive::favicon", instance_url = %instance_url, "Fetching favicon");

    // Parse the URL to get hostname and port
    let parsed_url = url::Url::parse(instance_url).context("Failed to parse instance URL")?;

    let host_with_port = if let Some(port) = parsed_url.port() {
        format!("{}:{}", parsed_url.host_str().unwrap_or(""), port)
    } else {
        parsed_url.host_str().unwrap_or("").to_string()
    };

    // Generate SHA256 hash of hostname:port
    let mut hasher = Sha256::new();
    hasher.update(host_with_port.as_bytes());
    let hash_hex = format!("{:x}", hasher.finalize());
    let hash = &hash_hex[..16];

    // Get icons directory
    let icons_dir = get_icons_dir()?;
    let ico_path = icons_dir.join(format!("{}.ico", hash));
    let raw_path = icons_dir.join(format!("{}_raw.png", hash));

    // Try to fetch and process icons from remote
    match fetch_icons_from_remote(instance_url, &icons_dir, hash).await {
        Ok(result) => return Ok(result),
        Err(e) => {
            tracing::warn!(target: "drive::favicon", error = %e, "Failed to fetch favicon from remote, using fallback");
        }
    }

    // Fallback: use bundled icons
    // Small icon fallback
    let small_bytes = get_fallback_icon(IconType::Small)?;
    save_icon(&small_bytes, &ico_path, false, true)?;
    tracing::debug!(target: "drive::favicon", path = %ico_path.display(), "Fallback ICO saved");

    // Large icon fallback
    let large_bytes = get_fallback_icon(IconType::Large)?;
    std::fs::write(&raw_path, &large_bytes).context("Failed to save fallback raw icon")?;
    tracing::debug!(target: "drive::favicon", path = %raw_path.display(), "Fallback raw icon saved");

    tracing::info!(target: "drive::favicon", ico_path = %ico_path.display(), raw_path = %raw_path.display(), "Fallback favicon saved successfully");

    Ok(FaviconResult {
        ico_path: ico_path.to_string_lossy().to_string(),
        raw_path: raw_path.to_string_lossy().to_string(),
    })
}

/// Fetch icons from remote server
async fn fetch_icons_from_remote(instance_url: &str, icons_dir: &PathBuf, hash: &str) -> Result<FaviconResult> {
    let client = reqwest::Client::builder()
        .timeout(std::time::Duration::from_secs(10))
        .build()
        .context("Failed to create HTTP client")?;

    // Fetch manifest.json
    let manifest_url = format!("{}/manifest.json", instance_url.trim_end_matches('/'));
    tracing::debug!(target: "drive::favicon", manifest_url = %manifest_url, "Fetching manifest.json");

    let manifest: Manifest = client
        .get(&manifest_url)
        .send()
        .await
        .context("Failed to fetch manifest.json")?
        .json()
        .await
        .context("Failed to parse manifest.json")?;

    // Find the smallest icon for ICO (Windows shell integration)
    let smallest_icon = manifest
        .icons
        .iter()
        .filter_map(|icon| parse_icon_size(&icon.sizes).map(|size| (size, icon)))
        .min_by_key(|(size, _)| *size)
        .map(|(_, icon)| icon)
        .context("No valid icons found in manifest")?;

    // Find the largest icon for raw image (status UI display)
    let largest_icon = manifest
        .icons
        .iter()
        .filter_map(|icon| parse_icon_size(&icon.sizes).map(|size| (size, icon)))
        .max_by_key(|(size, _)| *size)
        .map(|(_, icon)| icon)
        .unwrap_or(smallest_icon);

    tracing::debug!(target: "drive::favicon", smallest_src = %smallest_icon.src, smallest_sizes = %smallest_icon.sizes, "Selected smallest icon for ICO");
    tracing::debug!(target: "drive::favicon", largest_src = %largest_icon.src, largest_sizes = %largest_icon.sizes, "Selected largest icon for raw");

    let smallest_icon_url = build_icon_url(smallest_icon, instance_url);
    let largest_icon_url = build_icon_url(largest_icon, instance_url);
    let same_icon = largest_icon.src == smallest_icon.src;

    // Download smallest icon for ICO, with fallback
    let small_bytes = match download_icon(&client, &smallest_icon_url).await {
        Ok(bytes) => bytes.to_vec(),
        Err(e) => {
            tracing::warn!(target: "drive::favicon", error = %e, "Failed to download small icon, using fallback");
            get_fallback_icon(IconType::Small)?
        }
    };

    // Download largest icon for raw (or reuse if same), with fallback
    let large_bytes = if same_icon {
        small_bytes.clone()
    } else {
        match download_icon(&client, &largest_icon_url).await {
            Ok(bytes) => bytes.to_vec(),
            Err(e) => {
                tracing::warn!(target: "drive::favicon", error = %e, "Failed to download large icon, using fallback");
                get_fallback_icon(IconType::Large)?
            }
        }
    };

    // Determine paths and extensions
    let ico_path = icons_dir.join(format!("{}.ico", hash));
    let raw_extension = get_extension_from_icon(largest_icon, &largest_icon_url);
    let raw_path = icons_dir.join(format!("{}_raw.{}", hash, raw_extension));

    // Save small icon as ICO
    let is_already_ico = is_ico_format(smallest_icon, &smallest_icon_url);
    save_icon(&small_bytes, &ico_path, true, is_already_ico)?;
    tracing::debug!(target: "drive::favicon", path = %ico_path.display(), "ICO saved");

    // Save large icon as raw
    std::fs::write(&raw_path, &large_bytes).context("Failed to save raw icon file")?;
    tracing::debug!(target: "drive::favicon", path = %raw_path.display(), "Raw icon saved");

    tracing::info!(target: "drive::favicon", ico_path = %ico_path.display(), raw_path = %raw_path.display(), "Favicon saved successfully");

    Ok(FaviconResult {
        ico_path: ico_path.to_string_lossy().to_string(),
        raw_path: raw_path.to_string_lossy().to_string(),
    })
}
