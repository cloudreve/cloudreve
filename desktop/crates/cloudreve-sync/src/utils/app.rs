use std::sync::{Arc, OnceLock};
use windows::ApplicationModel;

static APP_ROOT: OnceLock<Arc<String>> = OnceLock::new();

pub fn init_app_root() {
    let path = ApplicationModel::Package::Current()
        .and_then(|p| p.InstalledLocation())
        .and_then(|l| l.Path())
        .map(|p| p.to_string())
        .unwrap_or_else(|_| String::new());

    APP_ROOT.set(Arc::new(path)).ok();
}

pub fn get_app_root() -> AppRoot {
    AppRoot(APP_ROOT.get().expect("APP_ROOT not initialized").clone())
}

#[derive(Clone)]
pub struct AppRoot(Arc<String>);

impl AppRoot {
    pub fn image_path(&self) -> String {
        match dark_light::detect().unwrap_or(dark_light::Mode::Light) {
            dark_light::Mode::Dark => format!("{}\\Images\\darkTheme", self.0.as_str()),
            dark_light::Mode::Light => format!("{}\\Images\\lightTheme", self.0.as_str()),
            dark_light::Mode::Unspecified => format!("{}\\Images\\lightTheme", self.0.as_str()),
        }
    }

    pub fn image_path_general(&self) -> String {
        format!("{}\\Images", self.0.as_str())
    }
}
