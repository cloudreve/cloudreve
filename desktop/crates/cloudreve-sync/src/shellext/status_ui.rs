use crate::drive::commands::ManagerCommand;
use crate::drive::manager::{DriveManager, DriveStatusUI, SyncStatus};
use crate::shellext::vector::create_vector;
use crate::utils::app::{AppRoot, get_app_root};
use std::sync::Arc;
use windows::Foundation::{EventRegistrationToken, TypedEventHandler, Uri};
use windows::{
    Storage::Provider::*,
    Win32::{Foundation::*, System::Com::*},
    core::*,
};
use tokio::sync::mpsc;

// UUID for our custom state handler - matches the C++ implementation
pub const CLSID_STATUS_UI_HANDLER: GUID = GUID::from_u128(0xb1d8ef74_822d_401a_a14a_25f45b1f70b7);

/// Different actions that can be triggered from the Status UI
#[derive(Clone)]
pub enum StatusUIAction {
    /// Show sync status (clicking on the sync status command)
    SyncStatus,
    /// Open user profile URL in browser
    OpenProfile { syncroot_id: String },
    /// Open storage/capacity details URL in browser
    OpenStorageDetails { syncroot_id: String },
    /// Open settings window
    OpenSettings,
}

#[implement(IStorageProviderUICommand)]
pub struct SyncStatusUICommand {
     #[allow(dead_code)]
    app_root: AppRoot,
    label: HSTRING,
    description: HSTRING,
    icon: Uri,
    action: StatusUIAction,
    command_tx: mpsc::UnboundedSender<ManagerCommand>,
}

impl SyncStatusUICommand {
    pub fn new(
        app_root: AppRoot,
        label: HSTRING,
        description: HSTRING,
        icon: Uri,
        action: StatusUIAction,
        command_tx: mpsc::UnboundedSender<ManagerCommand>,
    ) -> Self {
        Self {
            app_root,
            label,
            description,
            icon,
            action,
            command_tx,
        }
    }
}

impl IStorageProviderUICommand_Impl for SyncStatusUICommand_Impl {
    fn Label(&self) -> Result<HSTRING> {
        Ok(self.label.clone())
    }
    fn Description(&self) -> Result<HSTRING> {
        Ok(self.description.clone())
    }
    fn Icon(&self) -> Result<Uri> {
        Ok(self.icon.clone())
    }
    fn State(&self) -> Result<StorageProviderUICommandState> {
        Ok(StorageProviderUICommandState::Enabled)
    }
    fn Invoke(&self) -> Result<()> {
        tracing::debug!(target: "shellext::status_ui", "Invoke called");

        let command = match &self.action {
            StatusUIAction::SyncStatus => {
                tracing::debug!(target: "shellext::status_ui", "SyncStatus action - opening sync status window");
                ManagerCommand::OpenSyncStatusWindow
            }
            StatusUIAction::OpenProfile { syncroot_id } => {
                tracing::debug!(target: "shellext::status_ui", syncroot_id = %syncroot_id, "OpenProfile action");
                ManagerCommand::OpenProfileUrl { syncroot_id: syncroot_id.clone() }
            }
            StatusUIAction::OpenStorageDetails { syncroot_id } => {
                tracing::debug!(target: "shellext::status_ui", syncroot_id = %syncroot_id, "OpenStorageDetails action");
                ManagerCommand::OpenStorageDetailsUrl { syncroot_id: syncroot_id.clone() }
            }
            StatusUIAction::OpenSettings => {
                tracing::debug!(target: "shellext::status_ui", "OpenSettings action - opening settings window");
                ManagerCommand::OpenSettingsWindow
            }
        };

        if let Err(e) = self.command_tx.send(command) {
            tracing::error!(target: "shellext::status_ui", error = %e, "Failed to send command");
        }

        Ok(())
    }
}

#[implement(IStorageProviderStatusUISource)]
pub struct StatusUIHandler {
    drive_manager: Arc<DriveManager>,
    app_root: AppRoot,
    syncroot_id: String,
}

impl StatusUIHandler {
    pub fn new(drive_manager: Arc<DriveManager>, syncroot_id: String) -> Self {
        Self {
            drive_manager,
            app_root: get_app_root(),
            syncroot_id,
        }
    }

    /// Get drive status using the command pattern with blocking_recv
    fn get_drive_status(&self) -> Option<DriveStatusUI> {
        let command_tx = self.drive_manager.get_command_sender();
        let (response_tx, response_rx) = tokio::sync::oneshot::channel();

        if let Err(e) = command_tx.send(ManagerCommand::GetDriveStatusUI {
            syncroot_id: self.syncroot_id.clone(),
            response: response_tx,
        }) {
            tracing::error!(target: "shellext::status_ui", error = %e, "Failed to send GetDriveStatusUI command");
            return None;
        }

        match response_rx.blocking_recv() {
            Ok(Ok(status)) => status,
            Ok(Err(e)) => {
                tracing::error!(target: "shellext::status_ui", error = %e, "GetDriveStatusUI command failed");
                None
            }
            Err(e) => {
                tracing::error!(target: "shellext::status_ui", error = %e, "Failed to receive GetDriveStatusUI response");
                None
            }
        }
    }
}

impl IStorageProviderStatusUISource_Impl for StatusUIHandler_Impl {
    fn GetStatusUI(&self) -> Result<StorageProviderStatusUI> {
        tracing::trace!(target: "shellext::status_ui", syncroot_id = %self.syncroot_id, "GetStatusUI");

        let ui = StorageProviderStatusUI::new()?;
        let image_path = self.app_root.image_path();
        let command_tx = self.drive_manager.get_command_sender();

        // Get drive status from DriveManager
        let drive_status = self.get_drive_status();

        // Set provider state based on sync status
        let (provider_state, _state_label, sync_icon, sync_label, sync_description) = match &drive_status {
            Some(status) => {
                match status.sync_status {
                    SyncStatus::Syncing => (
                        StorageProviderState::Syncing,
                        status.name.clone(),
                        format!("{}\\CloudIconSyncing.svg", image_path),
                        t!("syncing").to_string(),
                        t!("syncingDescription", "count" => status.active_task_count).to_string(),
                    ),
                    SyncStatus::InSync => (
                        StorageProviderState::InSync,
                        status.name.clone(),
                        format!("{}\\CloudIconSynced.svg", image_path),
                        t!("synced").to_string(),
                        t!("syncedDescription").to_string(),
                    ),
                    SyncStatus::Paused => (
                        StorageProviderState::Paused,
                        status.name.clone(),
                        format!("{}\\CloudIconPaused.svg", image_path),
                        t!("paused").to_string(),
                        t!("pausedDescription").to_string(),
                    ),
                    SyncStatus::Error => (
                        StorageProviderState::Error,
                        status.name.clone(),
                        format!("{}\\CloudIconError.svg", image_path),
                        t!("error").to_string(),
                        t!("errorDescription").to_string(),
                    ),
                }
            }
            None => (
                StorageProviderState::InSync,
                "Cloudreve".to_string(),
                format!("{}\\CloudIconSynced.svg", image_path),
                t!("synced").to_string(),
                t!("syncedDescription").to_string(),
            ),
        };

        ui.SetProviderState(provider_state)?;

        ui.SetProviderStateLabel(&HSTRING::from("Cloudreve"))?;
        ui.SetProviderStateIcon(&Uri::CreateUri(&HSTRING::from(format!(
            "{}\\cloudreve.svg",
            image_path
        )))?)?;

        // Set sync status command - clicking shows the sync status window
        let sync_command: IStorageProviderUICommand = SyncStatusUICommand::new(
            self.app_root.clone(),
            HSTRING::from(&sync_label),
            HSTRING::from(&sync_description),
            Uri::CreateUri(&HSTRING::from(&sync_icon))?,
            StatusUIAction::SyncStatus,
            command_tx.clone(),
        )
        .into();
        ui.SetSyncStatusCommand(&sync_command)?;

        // Set primary command (capacity details) - only if capacity is available
        if let Some(ref status) = drive_status {
            if let Some(ref capacity) = status.capacity {
                let primary_command: IStorageProviderUICommand = SyncStatusUICommand::new(
                    self.app_root.clone(),
                    HSTRING::from(t!("capacityDetails").to_string()),
                    HSTRING::from(&capacity.label),
                    Uri::CreateUri(&HSTRING::from(format!(
                        "{}\\CloudIconSynced.svg",
                        image_path
                    )))?,
                    StatusUIAction::OpenStorageDetails { syncroot_id: self.syncroot_id.clone() },
                    command_tx.clone(),
                )
                .into();
                ui.SetProviderPrimaryCommand(&primary_command)?;
            }
        }

        // Set secondary commands (profile and settings) - only if status is available
        if let Some(ref status) = drive_status {
            let profile_command: IStorageProviderUICommand = SyncStatusUICommand::new(
                self.app_root.clone(),
                HSTRING::from(t!("profile").to_string()),
                HSTRING::from(&status.profile_url),
                Uri::CreateUri(&HSTRING::from(format!("{}\\ProfileIcon.svg", image_path)))?,
                StatusUIAction::OpenProfile { syncroot_id: self.syncroot_id.clone() },
                command_tx.clone(),
            )
            .into();

            let settings_command: IStorageProviderUICommand = SyncStatusUICommand::new(
                self.app_root.clone(),
                HSTRING::from(t!("settings").to_string()),
                HSTRING::from(&status.settings_url),
                Uri::CreateUri(&HSTRING::from(format!("{}\\SettingsIcon.svg", image_path)))?,
                StatusUIAction::OpenSettings,
                command_tx.clone(),
            )
            .into();

            let ivector = create_vector::<IStorageProviderUICommand>(vec![
                profile_command.into(),
                settings_command.into(),
            ])?;
            ui.SetProviderSecondaryCommands(&ivector)?;
        }

        // Set quota UI - only if capacity is available
        if let Some(ref status) = drive_status {
            if let Some(ref capacity) = status.capacity {
                let quota_ui = StorageProviderQuotaUI::new()?;
                quota_ui.SetQuotaUsedInBytes(capacity.used as u64)?;
                quota_ui.SetQuotaTotalInBytes(capacity.total as u64)?;
                quota_ui.SetQuotaUsedLabel(&HSTRING::from(&capacity.label))?;
                ui.SetQuotaUI(&quota_ui)?;
            }
        }

        Ok(ui)
    }

    fn StatusUIChanged(
        &self,
        handler: Option<
            &TypedEventHandler<IStorageProviderStatusUISource, windows_core::IInspectable>,
        >,
    ) -> windows_core::Result<EventRegistrationToken> {
        if let Some(handler) = handler {
            let _source: IStorageProviderStatusUISource = unsafe { self.this.cast()? };
            let handler = UIEvent(handler.clone());

            let _ = self.drive_manager.register_on_status_ui_changed(move || {
                tracing::trace!(target: "shellext::status_ui", "EventRegistrationToken: Invoking status UI changed callback");
                let _ = handler.Invoke(None, None);
            });
        }
        Ok(EventRegistrationToken::default())
    }

    fn RemoveStatusUIChanged(&self, _token: &EventRegistrationToken) -> windows_core::Result<()> {
        Ok(())
    }
}

#[implement(IStorageProviderStatusUISourceFactory)]
pub struct StatusUIHandlerFactory {
    drive_manager: Arc<DriveManager>,
}

impl StatusUIHandlerFactory {
    pub fn new(drive_manager: Arc<DriveManager>) -> Self {
        Self { drive_manager }
    }
}

impl IStorageProviderStatusUISourceFactory_Impl for StatusUIHandlerFactory_Impl {
    fn GetStatusUISource(&self, syncrootid: &HSTRING) -> Result<IStorageProviderStatusUISource> {
        let syncroot_id_str = syncrootid.to_string();
        tracing::trace!(target: "shellext::status_ui", syncroot_id = %syncroot_id_str, "GetStatusUISource");
        let handler = StatusUIHandler::new(self.drive_manager.clone(), syncroot_id_str);
        let handler: IStorageProviderStatusUISource = handler.into();
        Ok(handler)
    }
}

struct UIEvent(TypedEventHandler<IStorageProviderStatusUISource, windows_core::IInspectable>);
unsafe impl Send for UIEvent {}

impl UIEvent {
    #[allow(non_snake_case)]
    pub fn Invoke(
        &self,
        source: Option<&IStorageProviderStatusUISource>,
        args: Option<&IInspectable>,
    ) -> windows_core::Result<()> {
        self.0.Invoke(source, args)
    }
}

// Class factory for creating instances of our context menu handler
#[implement(IClassFactory)]
pub struct StatusUIHandlerFactoryFactory {
    drive_manager: Arc<DriveManager>,
}

impl StatusUIHandlerFactoryFactory {
    pub fn new(drive_manager: Arc<DriveManager>) -> Self {
        Self { drive_manager }
    }
}

impl IClassFactory_Impl for StatusUIHandlerFactoryFactory_Impl {
    fn CreateInstance(
        &self,
        outer: Option<&IUnknown>,
        iid: *const GUID,
        result: *mut *mut core::ffi::c_void,
    ) -> Result<()> {
        if outer.is_some() {
            return Err(Error::from(CLASS_E_NOAGGREGATION));
        }

        let handler = StatusUIHandlerFactory::new(self.drive_manager.clone());
        let handler: IUnknown = handler.into();

        unsafe { handler.query(iid, result).ok() }
    }

    fn LockServer(&self, _lock: BOOL) -> Result<()> {
        Ok(())
    }
}
