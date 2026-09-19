use std::sync::{Arc, OnceLock};
#[cfg(windows)]
use windows::ApplicationModel;

static APP_ROOT: OnceLock<Arc<String>> = OnceLock::new();

pub fn init_app_root() {
    // Windows shell integration loads icons and status UI resources from the
    // installed MSIX package root.
    #[cfg(windows)]
    let path = ApplicationModel::Package::Current()
        .and_then(|p| p.InstalledLocation())
        .and_then(|l| l.Path())
        .map(|p| p.to_string())
        .unwrap_or_else(|_| String::new());
    // Non-Windows builds do not use the Windows Explorer shell extension,
    // CFAPI placeholder UI, or MSIX package resources. Tauri and Linux
    // packaging handle their own UI/icon resources, so this remains an empty
    // compatibility value for shared code that expects APP_ROOT to be set.
    #[cfg(not(windows))]
    let path = String::new();

    APP_ROOT.set(Arc::new(path)).ok();
}

pub fn get_app_root() -> AppRoot {
    AppRoot(APP_ROOT.get().expect("APP_ROOT not initialized").clone())
}

#[derive(Clone)]
#[allow(dead_code)]
pub struct AppRoot(Arc<String>);

impl AppRoot {
    pub fn image_path(&self) -> String {
        #[cfg(windows)]
        {
            match dark_light::detect().unwrap_or(dark_light::Mode::Light) {
                dark_light::Mode::Dark => format!("{}\\Images\\darkTheme", self.0.as_str()),
                dark_light::Mode::Light => format!("{}\\Images\\lightTheme", self.0.as_str()),
                dark_light::Mode::Unspecified => {
                    format!("{}\\Images\\lightTheme", self.0.as_str())
                }
            }
        }
        #[cfg(not(windows))]
        {
            // Windows shell icons are not used by non-Windows full sync.
            String::new()
        }
    }

    pub fn image_path_general(&self) -> String {
        #[cfg(windows)]
        {
            format!("{}\\Images", self.0.as_str())
        }
        #[cfg(not(windows))]
        {
            // Windows shell icons are not used by non-Windows full sync.
            String::new()
        }
    }
}
