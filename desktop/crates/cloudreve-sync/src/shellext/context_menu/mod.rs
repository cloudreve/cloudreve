// Context menu handler for Windows Explorer
// This implements a COM object that provides a custom context menu item

mod explorer_command;
mod factory;
mod resolve_conflict;
mod sub_commands;
mod sync_now;
mod view_online;

pub use explorer_command::CrExplorerCommandHandler;
pub use factory::CrExplorerCommandFactory;
pub use resolve_conflict::ResolveConflictCommandHandler;
pub use sub_commands::SubCommands;
pub use sync_now::SyncNowCommandHandler;
pub use view_online::ViewOnlineCommandHandler;

use windows::ApplicationModel;
use windows::core::*;

// UUID for our context menu handler - matches the C++ implementation
pub const CLSID_EXPLORER_COMMAND: GUID = GUID::from_u128(0x165cd069_d9c8_42b4_8e37_b6971afa4494);

pub fn get_images_path() -> Result<String> {
    Ok(format!(
        "{}\\Images",
        ApplicationModel::Package::Current()?
            .InstalledLocation()?
            .Path()?
            .to_string(),
    ))
}
