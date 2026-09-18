use crate::drive::manager::DriveManager;
use crate::inventory::InventoryDb;
use crate::utils::app::{AppRoot, get_app_root};
use cloudreve_api::Boolset;
use cloudreve_api::models::explorer::file_permission;
use std::sync::Arc;
use windows::{
    Foundation::Collections::*,
    Storage::Provider::*,
    Win32::{Foundation::*, System::Com::*},
    core::*,
};

// UUID for our custom state handler - matches the C++ implementation
pub const CLSID_CUSTOM_STATE_HANDLER: GUID =
    GUID::from_u128(0xf0c9de6c_6c76_44d7_a58e_579cdf7af263);

#[implement(IStorageProviderItemPropertySource)]
pub struct CustomStateHandler {
     #[allow(dead_code)]
    drive_manager: Arc<DriveManager>,
    inventory: Arc<InventoryDb>,
    app_root: AppRoot,
}

impl CustomStateHandler {
    pub fn new(drive_manager: Arc<DriveManager>) -> Self {
        let inventory = drive_manager.get_inventory();
        Self {
            drive_manager,
            app_root: get_app_root(),
            inventory,
        }
    }
}

impl IStorageProviderItemPropertySource_Impl for CustomStateHandler_Impl {
    fn GetItemProperties(
        &self,
        itempath: &HSTRING,
    ) -> Result<IIterable<StorageProviderItemProperty>> {
        tracing::info!(target: "shellext::custom_state", "Getting item properties for {}", itempath);

        let file_metadata = self
            .inventory
            .query_by_path(itempath.to_string().as_str())
            .map_err(|e| {
                tracing::error!(target: "shellext::custom_state", "Failed to query inventory for path {}: {:?}", itempath, e);
                Error::from(E_FAIL)
            })?
            .ok_or_else(|| {
                tracing::error!(target: "shellext::custom_state", "No metadata found for path {}", itempath);
                Error::from(E_FAIL)
            })?;

        let image_path = self.app_root.image_path();
        let mut vec = Vec::new();

        if file_metadata.shared {
            let properties = StorageProviderItemProperty::new()?;
            properties.SetId(1)?;
            properties.SetIconResource(&HSTRING::from(format!("{}\\people.ico,0", image_path)))?;
            properties.SetValue(&HSTRING::from(t!("shared").as_ref()))?;
            vec.push(Some(properties));
        }

        if !file_metadata.permissions.is_empty() {
            let permission = Boolset::from_base64(&file_metadata.permissions).map_err(|e| {
                tracing::error!(target: "shellext::custom_state", "Failed to parse permission for path {}: {:?}", itempath, e);
                Error::from(E_FAIL)
            })?;
            if !permission.enabled(file_permission::READ as usize) {
                let properties = StorageProviderItemProperty::new()?;
                properties.SetId(2)?;
                properties
                    .SetIconResource(&HSTRING::from(format!("{}\\lock.ico,0", image_path)))?;
                properties.SetValue(&HSTRING::from(t!("noAccess").as_ref()))?;
                vec.push(Some(properties));
            }
        }

        IIterable::<StorageProviderItemProperty>::try_from(vec)
    }
}

// Class factory for creating instances of our context menu handler
#[implement(IClassFactory)]
pub struct CustomStateHandlerFactory {
    drive_manager: Arc<DriveManager>,
}

impl CustomStateHandlerFactory {
    pub fn new(drive_manager: Arc<DriveManager>) -> Self {
        Self { drive_manager }
    }
}

impl IClassFactory_Impl for CustomStateHandlerFactory_Impl {
    fn CreateInstance(
        &self,
        outer: Option<&IUnknown>,
        iid: *const GUID,
        result: *mut *mut core::ffi::c_void,
    ) -> Result<()> {
        if outer.is_some() {
            return Err(Error::from(CLASS_E_NOAGGREGATION));
        }

        let handler = CustomStateHandler::new(self.drive_manager.clone());
        let handler: IUnknown = handler.into();

        unsafe { handler.query(iid, result).ok() }
    }

    fn LockServer(&self, _lock: BOOL) -> Result<()> {
        Ok(())
    }
}
