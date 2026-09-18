use crate::drive::manager::DriveManager;
use crate::drive::sync::SyncMode;
use crate::{drive::commands::ManagerCommand, utils::app::AppRoot};
use rust_i18n::t;
use std::path::PathBuf;
use std::sync::Arc;
use windows::{
    Win32::{Foundation::*, System::Com::*, UI::Shell::*},
    core::*,
};

#[implement(IExplorerCommand)]
pub struct SyncNowCommandHandler {
    drive_manager: Arc<DriveManager>,
    app_root: AppRoot,
}

impl SyncNowCommandHandler {
    pub fn new(drive_manager: Arc<DriveManager>, app_root: AppRoot) -> Self {
        Self {
            drive_manager,
            app_root,
        }
    }
}

impl IExplorerCommand_Impl for SyncNowCommandHandler_Impl {
    fn GetTitle(&self, items: Option<&IShellItemArray>) -> Result<PWSTR> {
        let title = unsafe {
            match items {
                Some(items) => {
                    let count = items.GetCount()?;
                    if count == 0 {
                        t!("syncNow")
                    } else {
                        t!("syncSelectedNow")
                    }
                }
                None => t!("syncNow"),
            }
        };
        let hstring = HSTRING::from(title.as_ref());
        unsafe { SHStrDupW(&hstring) }
    }

    fn GetIcon(&self, _items: Option<&IShellItemArray>) -> Result<PWSTR> {
        let icon_path = format!("{}\\sync2.ico", self.app_root.image_path());
        let hstring = HSTRING::from(icon_path);
        unsafe { SHStrDupW(&hstring) }
    }

    fn GetToolTip(&self, _items: Option<&IShellItemArray>) -> Result<PWSTR> {
        Err(Error::from(E_NOTIMPL))
    }

    fn GetCanonicalName(&self) -> Result<GUID> {
        Ok(GUID::from_u128(0x50f8d185_47c9_45f8_a592_2d2cfefc9cd0))
    }

    fn GetState(&self, _items: Option<&IShellItemArray>, _oktobeslow: BOOL) -> Result<u32> {
        Ok(ECS_ENABLED.0 as u32)
    }

    fn Invoke(
        &self,
        selection: Option<&IShellItemArray>,
        _bindctx: Option<&IBindCtx>,
    ) -> Result<()> {
        tracing::debug!(target: "shellext::context_menu", "Sync now context menu command invoked");

        if let Some(items) = selection {
            unsafe {
                let count = items.GetCount()?;
                if count < 1 {
                    return Ok(());
                }

                let mut paths = Vec::new();
                for i in 0..count {
                    let item = items.GetItemAt(i)?;
                    let display_name = item.GetDisplayName(SIGDN_FILESYSPATH)?;
                    let path_str = display_name.to_string()?;
                    let path = PathBuf::from(path_str.clone());
                    paths.push(path);
                }

                // Send command through channel to async processor
                let command_tx = self.drive_manager.get_command_sender();
                if let Err(e) = command_tx.send(ManagerCommand::SyncNow {
                    paths: paths,
                    mode: SyncMode::FullHierarchy,
                }) {
                    tracing::error!(target: "shellext::context_menu", error = %e, "Failed to send SyncNow command");
                }
            }
        }

        Ok(())
    }

    fn GetFlags(&self) -> Result<u32> {
        Ok(ECF_DEFAULT.0 as u32)
    }

    fn EnumSubCommands(&self) -> Result<IEnumExplorerCommand> {
        Err(Error::from(E_NOTIMPL))
    }
}
