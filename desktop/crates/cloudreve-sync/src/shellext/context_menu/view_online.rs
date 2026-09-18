use crate::{drive::commands::ManagerCommand, utils::app::AppRoot};
use crate::drive::manager::DriveManager;
use rust_i18n::t;
use std::path::PathBuf;
use std::sync::Arc;
use windows::{
    Win32::{Foundation::*, System::Com::*, UI::Shell::*},
    core::*,
};

#[implement(IExplorerCommand)]
pub struct ViewOnlineCommandHandler {
    drive_manager: Arc<DriveManager>,
    app_root: AppRoot,

    #[allow(dead_code)]
    site: Option<IUnknown>,
}

impl ViewOnlineCommandHandler {
    pub fn new(drive_manager: Arc<DriveManager>, app_root: AppRoot) -> Self {
        Self {
            drive_manager,
            app_root,
            site: None,
        }
    }
}

impl IExplorerCommand_Impl for ViewOnlineCommandHandler_Impl {
    fn GetTitle(&self, _items: Option<&IShellItemArray>) -> Result<PWSTR> {
        let title = t!("viewOnline");
        let hstring = HSTRING::from(title.as_ref());
        unsafe { SHStrDupW(&hstring) }
    }

    fn GetIcon(&self, _items: Option<&IShellItemArray>) -> Result<PWSTR> {
        let icon_path = format!("{}\\globe7.ico", self.app_root.image_path());
        let hstring = HSTRING::from(icon_path);
        unsafe { SHStrDupW(&hstring) }
    }

    fn GetToolTip(&self, _items: Option<&IShellItemArray>) -> Result<PWSTR> {
        Err(Error::from(E_NOTIMPL))
    }

    fn GetCanonicalName(&self) -> Result<GUID> {
        tracing::trace!(target: "shellext::context_menu:view_online", "GetCanonicalName called");
        Ok(GUID::from_u128(0xe9206944_a659_434b_967b_27e15d2fef20))
    }

    fn GetState(&self, items: Option<&IShellItemArray>, _oktobeslow: BOOL) -> Result<u32> {
        let Some(items) = items else {
            // Not select anthing, but still triggerd from a folder
            return Ok(ECS_ENABLED.0 as u32);
        };

        unsafe {
            let count = items.GetCount()?;
            if count <= 1 {
                Ok(ECS_ENABLED.0 as u32)
            } else {
                Ok(ECS_HIDDEN.0 as u32)
            }
        }
    }

    fn Invoke(
        &self,
        selection: Option<&IShellItemArray>,
        _bindctx: Option<&IBindCtx>,
    ) -> Result<()> {
        tracing::debug!(target: "shellext::context_menu", "View online context menu command invoked");

        if let Some(items) = selection {
            unsafe {
                let count = items.GetCount()?;
                if count != 1 {
                    return Ok(());
                }

                // Get the first item
                let item = items.GetItemAt(0)?;
                let display_name = item.GetDisplayName(SIGDN_FILESYSPATH)?;
                let path_str = display_name.to_string()?;
                let path = PathBuf::from(path_str.clone());

                tracing::debug!(target: "shellext::context_menu", path = %path_str, "View online requested");

                // Send command through channel to async processor
                let command_tx = self.drive_manager.get_command_sender();

                if let Err(e) = command_tx.send(ManagerCommand::ViewOnline { path: path.clone() }) {
                    tracing::error!(target: "shellext::context_menu", error = %e, "Failed to send ViewOnline command");
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
