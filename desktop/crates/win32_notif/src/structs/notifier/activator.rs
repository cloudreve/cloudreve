use windows::core::implement;
use windows::Win32::UI::Notifications::{
  INotificationActivationCallback, INotificationActivationCallback_Impl,
  NOTIFICATION_USER_INPUT_DATA,
};

#[implement(INotificationActivationCallback)]
pub struct ToastActivationManager;

impl INotificationActivationCallback_Impl for ToastActivationManager_Impl {
  fn Activate(
    &self,
    _appusermodelid: &windows_core::PCWSTR,
    invokedargs: &windows_core::PCWSTR,
    _data: *const NOTIFICATION_USER_INPUT_DATA,
    _count: u32,
  ) -> windows_core::Result<()> {
    println!("Called");

    if invokedargs.is_null() {
      println!("Toast activated (default click), no arguments.");
    } else {
      // Convert invoked_args (LPCWSTR) to a Rust String/str
      let args_str = unsafe { invokedargs.to_string() };
      println!(
        "Toast activated with arguments: {}",
        args_str.unwrap_or_default()
      );
    }

    Ok(())
  }
}
