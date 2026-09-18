use windows::{
  core::{Error, Interface, Ref},
  Foundation::TypedEventHandler,
  UI::Notifications::{ToastFailedEventArgs, ToastNotification},
};

use crate::notification::PartialNotification;

#[derive(Debug)]
pub struct ToastFailedArgs {
  pub error: Option<String>,
}

impl ToastFailedArgs {
  pub(crate) fn new(args: ToastFailedEventArgs) -> Self {
    Self {
      error: args.ErrorCode().ok().and_then(|x| x.to_string().into()),
    }
  }
}

pub struct NotificationFailedEventHandler {
  pub(crate) handler: TypedEventHandler<ToastNotification, ToastFailedEventArgs>,
}

impl NotificationFailedEventHandler {
  pub fn new<
    T: Fn(Option<PartialNotification>, Option<ToastFailedArgs>) -> Result<(), Error>
      + Send
      + Sync
      + 'static,
  >(
    func: T,
  ) -> Self {
    let handler: TypedEventHandler<ToastNotification, ToastFailedEventArgs> =
      TypedEventHandler::new(
        move |a: Ref<ToastNotification>, b: Ref<ToastFailedEventArgs>| {
          let a = a.as_ref();
          let a = a.and_then(|a| PartialNotification { _toast: a }.into());

          let b = b.as_ref();
          let b = b.and_then(|x| x.cast::<ToastFailedEventArgs>().ok());
          let b = b.and_then(|x| Some(ToastFailedArgs::new(x)));

          func(a, b)
        },
      );

    Self { handler }
  }
}
