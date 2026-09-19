use std::path::PathBuf;

use anyhow::{Context, Result};
use cloudreve_api::models::uri::CrUri;
use url::Url;
#[cfg(windows)]
use widestring::U16CString;
#[cfg(windows)]
use windows::Win32::UI::Shell::{SHCNE_ID, SHCNF_PATHW, SHChangeNotify};

use crate::drive::mounts::DriveConfig;

pub fn local_path_to_cr_uri(path: PathBuf, root: PathBuf, remote_base: String) -> Result<CrUri> {
    let mut base = CrUri::new(&remote_base)?;

    // Strip the root from path to get the relative path
    let relative = path.strip_prefix(&root).context("Path is not under root")?;

    // Convert to string with forward slashes (for URI compatibility)
    let relative_str = relative
        .to_str()
        .context("Path contains invalid UTF-8")?
        .replace("\\", "/");

    // Join the relative path to the base URI if not empty
    if !relative_str.is_empty() {
        base.join(&relative_str.split("/").collect::<Vec<&str>>());
    }

    Ok(base)
}

pub fn remote_path_to_local_relative_path(
    remote_path: &CrUri,
    remote_base: &CrUri,
) -> Result<PathBuf> {
    let remote_path_str = remote_path.path().clone();
    let remote_base_str = remote_base.path().clone();

    // 1. add ending slash if not presented to remote_base_str
    let remote_base_str = if !remote_base_str.ends_with('/') {
        remote_base_str + "/"
    } else {
        remote_base_str
    };

    // 2. remove remote_base_str from remote_path_str
    let relative_path = remote_path_str
        .strip_prefix(&remote_base_str)
        .context("Path is not under remote base")?;

    // 3. make sure OS slash is used
    let relative_path = relative_path.replace("/", std::path::MAIN_SEPARATOR_STR);

    Ok(PathBuf::from(relative_path))
}

/// Generate a URL to view a folder or file online.
///
/// For folders: pass the folder path as `folder_path` and None for `open_file`
/// For files: pass the parent folder path as `folder_path` and the file path as `open_file`
pub fn view_online_url(
    folder_path: &str,
    open_file: Option<&str>,
    config: &DriveConfig,
) -> Result<String> {
    let mut base = config.instance_url.parse::<Url>()?;
    base.set_path("/home");

    {
        let mut query = base.query_pairs_mut();
        query.append_pair("path", folder_path);

        if let Some(file) = open_file {
            query.append_pair("open", file);
        }

        query.append_pair("user_hint", config.user_id.as_str());
    }

    Ok(base.to_string())
}

pub fn recycle_bin_url(config: &DriveConfig) -> Result<String> {
    let mut base = config.instance_url.parse::<Url>()?;
    base.set_path("/home");

    {
        let mut query = base.query_pairs_mut();
        query.append_pair("user_hint", config.user_id.as_str());
        query.append_pair("path", "cloudreve://trash");
    }

    Ok(base.to_string())
}

/// Notifies the platform shell that a local file or directory changed.
///
/// On Windows this calls `SHChangeNotify` so Explorer refreshes Cloud Files
/// placeholder state, shell overlays, and file attributes after create/delete or
/// hydrate/dehydrate operations.
#[cfg(windows)]
pub fn notify_shell_change(path: &PathBuf, event: SHCNE_ID) -> Result<()> {
    let utf16_path = U16CString::from_os_str(path.as_path())?;
    unsafe {
        SHChangeNotify(
            event,
            SHCNF_PATHW,
            Some(utf16_path.as_ptr() as *const _),
            None,
        );
    }
    Ok(())
}

/// No-op shell notification shim for non-Windows full sync.
///
/// Linux and other non-Windows platforms do not use the Windows CFAPI
/// placeholder model, and there is no single cross-desktop equivalent of
/// `SHChangeNotify`. Ordinary file managers observe real file and directory
/// changes through filesystem notifications, so full-sync correctness does not
/// depend on an explicit shell refresh hook here.
#[cfg(not(windows))]
pub fn notify_shell_change(_path: &PathBuf, _event: u32) -> Result<()> {
    Ok(())
}
