use crate::drive::commands::{ConflictAction, ManagerCommand};
use crate::drive::manager::DriveManager;
use crate::inventory::InventoryDb;
use crate::utils::app::{AppRoot, get_app_root};
use std::collections::HashMap;
use base64::{Engine as _, engine::general_purpose::URL_SAFE};
use std::sync::Arc;
use windows::{
    Win32::{Foundation::*, System::Com::*, UI::Notifications::*},
    core::*,
};

pub const CLSID_TOAST_ACTIVATOR: GUID = GUID::from_u128(0xeffe04d9_151d_49da_9eb5_34e01442edfe);

/// Represents the parsed action from toast notification arguments
#[derive(Debug, Clone)]
pub struct ToastAction {
    /// The action type (e.g., "resolve", "dismiss", "viewImage")
    pub action: String,
    /// Additional parameters parsed from the arguments
    pub params: HashMap<String, String>,
}

impl ToastAction {
    /// Parse toast arguments string into a ToastAction
    /// Supports formats like:
    /// - "action=resolve"
    /// - "action=resolve&file_id=123&drive_id=456"
    /// - "resolve" (action name only)
    pub fn parse(args: &str) -> Self {
        let mut action = String::new();
        let mut params = HashMap::new();

        // Split by '&' to get key=value pairs
        for part in args.split('&') {
            if let Some((key, value)) = part.split_once('=') {
                if key == "action" {
                    action = value.to_string();
                } else {
                    params.insert(key.to_string(), value.to_string());
                }
            } else if action.is_empty() {
                // If no '=' found and action is empty, treat the whole part as the action
                action = part.to_string();
            }
        }

        Self { action, params }
    }
}

/// User input data from toast notification
#[derive(Debug, Clone)]
pub struct ToastInputData {
    pub key: String,
    pub value: String,
}

#[implement(INotificationActivationCallback)]
pub struct ToastActivator {
    drive_manager: Arc<DriveManager>,
     #[allow(dead_code)]
    inventory: Arc<InventoryDb>,
     #[allow(dead_code)]
    app_root: AppRoot,
}

impl ToastActivator {
    pub fn new(drive_manager: Arc<DriveManager>) -> Self {
        let inventory = drive_manager.get_inventory();
        Self {
            drive_manager,
            app_root: get_app_root(),
            inventory,
        }
    }

    /// Parse the NOTIFICATION_USER_INPUT_DATA array into a Vec of ToastInputData
    fn parse_input_data(
        data: *const NOTIFICATION_USER_INPUT_DATA,
        count: u32,
    ) -> Vec<ToastInputData> {
        let mut inputs = Vec::new();

        if data.is_null() || count == 0 {
            return inputs;
        }

        unsafe {
            let slice = std::slice::from_raw_parts(data, count as usize);
            for input in slice {
                let key = input.Key.to_string().unwrap_or_default();
                let value = input.Value.to_string().unwrap_or_default();
                inputs.push(ToastInputData { key, value });
            }
        }

        inputs
    }

    /// Handle the resolve action for conflict resolution
    fn handle_resolve_action(&self, inputs: &[ToastInputData], params: &HashMap<String, String>) {
        // Get the first input data (selection value)
        if let Some(first_input) = inputs.first() {
            tracing::info!(
                key = %first_input.key,
                value = %first_input.value,
                ?params,
                "Handling resolve action"
            );

            let conflict_action =
                ConflictAction::from_str(&first_input.value).unwrap_or(ConflictAction::KeepRemote);
            let command_tx = self.drive_manager.get_command_sender();
            if let Err(e) = command_tx.send(ManagerCommand::ResolveConflict {
                drive_id: params.get("drive_id").unwrap_or(&String::new()).to_string(),
                path: URL_SAFE.decode(params.get("path").unwrap_or(&String::new()).to_string().as_bytes())
                    .ok()
                    .and_then(|bytes| String::from_utf8(bytes).ok())
                    .unwrap_or_default(),
                file_id: params
                    .get("file_id")
                    .unwrap_or(&String::new())
                    .parse::<i64>()
                    .unwrap_or(0),
                action: conflict_action,
            }) {
                tracing::error!(error = ?e, "Failed to send ResolveConflict command");
            }
        }
    }

    /// Handle the settings action to open settings window
    fn handle_settings_action(&self, params: &HashMap<String, String>) {
        tracing::debug!(?params, "Opening settings window from toast");
        let command_tx = self.drive_manager.get_command_sender();
        if let Err(e) = command_tx.send(ManagerCommand::OpenSettingsWindow) {
            tracing::error!(error = ?e, "Failed to send OpenSettingsWindow command");
        }
    }

    /// Handle the dismiss action
    fn handle_dismiss_action(&self, params: &HashMap<String, String>) {
        tracing::debug!(?params, "Toast dismissed by user");
        // Dismiss is usually a no-op, but we could log or track dismissals
    }

    /// Handle opening the app window (foreground activation)
    fn handle_foreground_activation(&self, params: &HashMap<String, String>) {
        tracing::debug!(?params, "Foreground activation - opening app window");
        // TODO: Implement window opening logic if needed
        // This could be used when user clicks on the toast body itself
    }
}

impl INotificationActivationCallback_Impl for ToastActivator_Impl {
    fn Activate(
        &self,
        appusermodelid: &windows_core::PCWSTR,
        invokedargs: &windows_core::PCWSTR,
        data: *const NOTIFICATION_USER_INPUT_DATA,
        count: u32,
    ) -> windows_core::Result<()> {
        tracing::trace!(
            "Toast activated: appusermodelid={:?}, invokedargs={:?}, data={:?}, count={}",
            appusermodelid,
            invokedargs,
            data,
            count
        );

        // Parse the invoked arguments
        let args = unsafe {
            if invokedargs.is_null() {
                return Ok(());
            }
            match invokedargs.to_string() {
                Ok(s) => s,
                Err(e) => {
                    tracing::error!(?e, "Failed to parse invoked arguments");
                    return Ok(());
                }
            }
        };

        if args.is_empty() {
            // Empty args typically means user clicked on the toast body itself
            self.handle_foreground_activation(&HashMap::new());
            return Ok(());
        }

        tracing::debug!(?args, "Toast activated with arguments");

        // Parse the action and parameters
        let toast_action = ToastAction::parse(&args);
        tracing::debug!(?toast_action, "Parsed toast action");

        // Parse the input data
        let inputs = ToastActivator::parse_input_data(data, count);
        if !inputs.is_empty() {
            tracing::debug!(?inputs, "Received user input data");
        }

        // Handle different actions
        match toast_action.action.as_str() {
            "resolve" => {
                self.handle_resolve_action(&inputs, &toast_action.params);
            }
            "dismiss" => {
                self.handle_dismiss_action(&toast_action.params);
            }
            "settings" => {
                // Open settings window
                self.handle_settings_action(&toast_action.params);
            }
            "" => {
                // Empty action - foreground activation (user clicked on toast body)
                self.handle_foreground_activation(&toast_action.params);
            }
            other => {
                tracing::warn!(action = %other, "Unknown toast action");
                // For unknown actions, treat as foreground activation
                self.handle_foreground_activation(&toast_action.params);
            }
        }

        Ok(())
    }
}

// Class factory for creating instances of our toast activator
#[implement(IClassFactory)]
pub struct ToastActivatorFactory {
    drive_manager: Arc<DriveManager>,
}

impl ToastActivatorFactory {
    pub fn new(drive_manager: Arc<DriveManager>) -> Self {
        Self { drive_manager }
    }
}

impl IClassFactory_Impl for ToastActivatorFactory_Impl {
    fn CreateInstance(
        &self,
        outer: Option<&IUnknown>,
        iid: *const GUID,
        result: *mut *mut core::ffi::c_void,
    ) -> Result<()> {
        if outer.is_some() {
            return Err(Error::from(CLASS_E_NOAGGREGATION));
        }

        let handler = ToastActivator::new(self.drive_manager.clone());
        let handler: IUnknown = handler.into();

        unsafe { handler.query(iid, result).ok() }
    }

    fn LockServer(&self, _lock: BOOL) -> Result<()> {
        Ok(())
    }
}
