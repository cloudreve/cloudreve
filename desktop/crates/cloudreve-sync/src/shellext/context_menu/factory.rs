use super::CrExplorerCommandHandler;
use crate::drive::manager::DriveManager;
use std::sync::Arc;
use windows::{
    Win32::{Foundation::*, System::Com::*},
    core::*,
};

// Class factory for creating instances of our context menu handler
#[implement(IClassFactory)]
pub struct CrExplorerCommandFactory {
    drive_manager: Arc<DriveManager>,
}

impl CrExplorerCommandFactory {
    pub fn new(drive_manager: Arc<DriveManager>) -> Self {
        Self { drive_manager }
    }
}

impl IClassFactory_Impl for CrExplorerCommandFactory_Impl {
    fn CreateInstance(
        &self,
        outer: Option<&IUnknown>,
        iid: *const GUID,
        result: *mut *mut core::ffi::c_void,
    ) -> Result<()> {
        if outer.is_some() {
            return Err(Error::from(CLASS_E_NOAGGREGATION));
        }

        let handler = CrExplorerCommandHandler::new(self.drive_manager.clone());
        let handler: IUnknown = handler.into();

        unsafe { handler.query(iid, result).ok() }
    }

    fn LockServer(&self, _lock: BOOL) -> Result<()> {
        Ok(())
    }
}
