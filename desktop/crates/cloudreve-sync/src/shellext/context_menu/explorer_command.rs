use super::{CLSID_EXPLORER_COMMAND, SubCommands};
use crate::{drive::manager::DriveManager, utils::app::{AppRoot, get_app_root}};
use std::sync::Arc;
use windows::{
    Win32::{Foundation::*, System::Com::*, UI::Shell::*},
    core::*,
};

#[implement(IExplorerCommand)]
pub struct CrExplorerCommandHandler {
    drive_manager: Arc<DriveManager>,
    app_root: AppRoot,

    #[allow(dead_code)]
    site: std::sync::Mutex<Option<IUnknown>>,
}

impl CrExplorerCommandHandler {
    pub fn new(drive_manager: Arc<DriveManager>) -> Self {
        Self {
            drive_manager: drive_manager.clone(),
            app_root: get_app_root(),
            site: std::sync::Mutex::new(None),
        }
    }
}

impl IExplorerCommand_Impl for CrExplorerCommandHandler_Impl {
    fn GetTitle(&self, _items: Option<&IShellItemArray>) -> Result<PWSTR> {
        let hstring = HSTRING::from("Cloudreve");
        unsafe { SHStrDupW(&hstring) }
    }

    fn GetIcon(&self, _items: Option<&IShellItemArray>) -> Result<PWSTR> {
        let icon_path = format!("{}\\cloudreve.ico", self.app_root.image_path_general());
        let hstring = HSTRING::from(icon_path);
        unsafe { SHStrDupW(&hstring) }
    }

    fn GetToolTip(&self, _items: Option<&IShellItemArray>) -> Result<PWSTR> {
        Err(Error::from(E_NOTIMPL))
    }

    fn GetCanonicalName(&self) -> Result<GUID> {
        Ok(CLSID_EXPLORER_COMMAND)
    }

    fn GetState(&self, _items: Option<&IShellItemArray>, _oktobeslow: BOOL) -> Result<u32> {
        Ok(ECS_ENABLED.0 as u32)
    }

    fn Invoke(
        &self,
        _selection: Option<&IShellItemArray>,
        _bindctx: Option<&IBindCtx>,
    ) -> Result<()> {
        tracing::debug!(target: "shellext::context_menu", "View online context menu command invoked");
        Ok(())
    }

    fn GetFlags(&self) -> Result<u32> {
        Ok((ECF_DEFAULT.0 | ECF_HASSUBCOMMANDS.0 | ECF_ISDROPDOWN.0) as u32)
    }

    fn EnumSubCommands(&self) -> Result<IEnumExplorerCommand> {
        tracing::trace!(target: "shellext::context_menu", "EnumSubCommands called");
        Ok(SubCommands::new(self.drive_manager.clone(), self.app_root.clone()).into())
    }
}
