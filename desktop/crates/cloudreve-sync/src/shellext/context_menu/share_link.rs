use crate::drive::manager::DriveManager;
use crate::{drive::commands::ManagerCommand, utils::app::AppRoot};
use rust_i18n::t;
use std::path::PathBuf;
use std::sync::{Arc, Mutex};
use windows::{
    Win32::{Foundation::*, System::Com::*, System::Ole::*, UI::Shell::*},
    core::*,
};

/// shlguid.h: SID_SFolderView — resolves IFolderView from the command site.
const SID_S_FOLDER_VIEW: GUID = GUID::from_u128(0xcde725b0_ccc9_4519_917e_325d72fab4ce);

#[implement(IExplorerCommand, IObjectWithSite)]
pub struct ShareLinkCommandHandler {
    drive_manager: Arc<DriveManager>,
    app_root: AppRoot,
    site: Mutex<Option<IUnknown>>,
}

impl ShareLinkCommandHandler {
    pub fn new(drive_manager: Arc<DriveManager>, app_root: AppRoot) -> Self {
        Self {
            drive_manager,
            app_root,
            site: Mutex::new(None),
        }
    }

    fn send_copy_share_link(&self, path: PathBuf) {
        tracing::debug!(target: "shellext::context_menu", path = %path.display(), "Copy share link requested");
        let command_tx = self.drive_manager.get_command_sender();
        if let Err(e) = command_tx.send(ManagerCommand::CopyShareLink { path }) {
            tracing::error!(target: "shellext::context_menu", error = %e, "Failed to send CopyShareLink command");
        }
    }
}

impl IObjectWithSite_Impl for ShareLinkCommandHandler_Impl {
    fn SetSite(&self, punksite: Option<&IUnknown>) -> Result<()> {
        *self.site.lock().unwrap() = punksite.cloned();
        Ok(())
    }

    fn GetSite(&self, riid: *const GUID, ppvsite: *mut *mut core::ffi::c_void) -> Result<()> {
        let site = self.site.lock().unwrap();
        if let Some(site) = site.as_ref() {
            unsafe { site.query(riid, ppvsite) }.ok()
        } else {
            Err(Error::from(E_FAIL))
        }
    }
}

impl IExplorerCommand_Impl for ShareLinkCommandHandler_Impl {
    fn GetTitle(&self, _items: Option<&IShellItemArray>) -> Result<PWSTR> {
        let title = t!("copyShareLink");
        let hstring = HSTRING::from(title.as_ref());
        unsafe { SHStrDupW(&hstring) }
    }

    fn GetIcon(&self, _items: Option<&IShellItemArray>) -> Result<PWSTR> {
        let icon_path = format!("{}\\people.ico", self.app_root.image_path());
        let hstring = HSTRING::from(icon_path);
        unsafe { SHStrDupW(&hstring) }
    }

    fn GetToolTip(&self, _items: Option<&IShellItemArray>) -> Result<PWSTR> {
        Err(Error::from(E_NOTIMPL))
    }

    fn GetCanonicalName(&self) -> Result<GUID> {
        Ok(GUID::from_u128(0x7d2b8f1c_3a9e_4c5d_b6f2_9e8a1d4c6f0b))
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
        tracing::debug!(target: "shellext::context_menu", "Copy share link context menu command invoked");

        if let Some(items) = selection {
            unsafe {
                let count = items.GetCount()?;
                if count != 1 {
                    return Ok(());
                }

                // Get the first item
                let item = items.GetItemAt(0)?;
                let display_name = item.GetDisplayName(SIGDN_FILESYSPATH)?;
                let path = PathBuf::from(display_name.to_string()?);

                self.send_copy_share_link(path);
            }
        } else {
            // Folder-background invoke carries no selection; resolve the
            // current folder through the site Explorer gave us in SetSite.
            let site = self.site.lock().unwrap().clone();
            let Some(site) = site else {
                return Ok(());
            };
            unsafe {
                let service_provider: IServiceProvider = site.cast()?;
                let folder_view: IFolderView =
                    service_provider.QueryService(&SID_S_FOLDER_VIEW)?;
                let item: IShellItem = folder_view.GetFolder()?;
                let display_name = item.GetDisplayName(SIGDN_FILESYSPATH)?;
                let path = PathBuf::from(display_name.to_string()?);

                self.send_copy_share_link(path);
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
