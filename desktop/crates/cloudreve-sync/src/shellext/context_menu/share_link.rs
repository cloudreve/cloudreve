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

/// Resolve the invoked filesystem path from either the selection or, for
/// folder-background invokes, the folder behind the command site.
fn resolve_invoke_path(
    selection: Option<&IShellItemArray>,
    site: &Mutex<Option<IUnknown>>,
) -> Result<Option<PathBuf>> {
    if let Some(items) = selection {
        unsafe {
            let count = items.GetCount()?;
            if count != 1 {
                return Ok(None);
            }

            let item = items.GetItemAt(0)?;
            let display_name = item.GetDisplayName(SIGDN_FILESYSPATH)?;
            return Ok(Some(PathBuf::from(display_name.to_string()?)));
        }
    }

    // Folder-background invoke carries no selection; resolve the current
    // folder through the site Explorer gave us in SetSite.
    let site = site.lock().unwrap().clone();
    let Some(site) = site else {
        return Ok(None);
    };
    unsafe {
        let service_provider: IServiceProvider = site.cast()?;
        let folder_view: IFolderView = service_provider.QueryService(&SID_S_FOLDER_VIEW)?;
        let item: IShellItem = folder_view.GetFolder()?;
        let display_name = item.GetDisplayName(SIGDN_FILESYSPATH)?;
        Ok(Some(PathBuf::from(display_name.to_string()?)))
    }
}

fn item_count_state(items: Option<&IShellItemArray>) -> Result<u32> {
    let Some(items) = items else {
        // Not selecting anything, but still triggered from a folder
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

macro_rules! share_command_handler {
    ($name:ident, $impl_name:ident, $title_key:literal, $icon:literal, $guid:literal, $variant:ident) => {
        #[implement(IExplorerCommand, IObjectWithSite)]
        pub struct $name {
            drive_manager: Arc<DriveManager>,
            app_root: AppRoot,
            site: Mutex<Option<IUnknown>>,
        }

        impl $name {
            pub fn new(drive_manager: Arc<DriveManager>, app_root: AppRoot) -> Self {
                Self {
                    drive_manager,
                    app_root,
                    site: Mutex::new(None),
                }
            }
        }

        impl IObjectWithSite_Impl for $impl_name {
            fn SetSite(&self, punksite: Option<&IUnknown>) -> Result<()> {
                *self.site.lock().unwrap() = punksite.cloned();
                Ok(())
            }

            fn GetSite(
                &self,
                riid: *const GUID,
                ppvsite: *mut *mut core::ffi::c_void,
            ) -> Result<()> {
                let site = self.site.lock().unwrap();
                if let Some(site) = site.as_ref() {
                    unsafe { site.query(riid, ppvsite) }.ok()
                } else {
                    Err(Error::from(E_FAIL))
                }
            }
        }

        impl IExplorerCommand_Impl for $impl_name {
            fn GetTitle(&self, _items: Option<&IShellItemArray>) -> Result<PWSTR> {
                let title = t!($title_key);
                let hstring = HSTRING::from(title.as_ref());
                unsafe { SHStrDupW(&hstring) }
            }

            fn GetIcon(&self, _items: Option<&IShellItemArray>) -> Result<PWSTR> {
                let icon_path = format!("{}\\{}", self.app_root.image_path(), $icon);
                let hstring = HSTRING::from(icon_path);
                unsafe { SHStrDupW(&hstring) }
            }

            fn GetToolTip(&self, _items: Option<&IShellItemArray>) -> Result<PWSTR> {
                Err(Error::from(E_NOTIMPL))
            }

            fn GetCanonicalName(&self) -> Result<GUID> {
                Ok(GUID::from_u128($guid))
            }

            fn GetState(
                &self,
                items: Option<&IShellItemArray>,
                _oktobeslow: BOOL,
            ) -> Result<u32> {
                item_count_state(items)
            }

            fn Invoke(
                &self,
                selection: Option<&IShellItemArray>,
                _bindctx: Option<&IBindCtx>,
            ) -> Result<()> {
                tracing::debug!(target: "shellext::context_menu", concat!($title_key, " context menu command invoked"));

                if let Some(path) = resolve_invoke_path(selection, &self.site)? {
                    let command_tx = self.drive_manager.get_command_sender();
                    if let Err(e) = command_tx.send(ManagerCommand::$variant { path }) {
                        tracing::error!(target: "shellext::context_menu", error = %e, concat!("Failed to send ", $title_key, " command"));
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
    };
}

share_command_handler!(
    ShareLinkCommandHandler,
    ShareLinkCommandHandler_Impl,
    "shareLink",
    "people.ico",
    0x7d2b8f1c_3a9e_4c5d_b6f2_9e8a1d4c6f0b,
    ShareLink
);

share_command_handler!(
    CopyShareLinkCommandHandler,
    CopyShareLinkCommandHandler_Impl,
    "copyShareLink",
    "globe7.ico",
    0x4a1f9e2d_8b7c_4d3e_a5f1_2c7b9d8e6a4f,
    CopyShareLink
);
