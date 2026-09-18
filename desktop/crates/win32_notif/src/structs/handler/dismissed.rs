use windows::{
  core::{Error, Interface, Ref},
  Foundation::TypedEventHandler,
  UI::Notifications::{ToastDismissalReason, ToastDismissedEventArgs, ToastNotification},
};

use crate::notification::PartialNotification;

#[derive(Debug)]
pub enum ToastDismissedReason {
  Unknown(String),
  UserCanceled,
  ApplicationHidden,
  TimedOut,
}

impl ToastDismissedReason {
  pub(crate) fn new(args: ToastDismissedEventArgs) -> Self {
    args.Reason().map_or_else(
      |x| Self::Unknown(x.message()),
      |x| {
        let x = x.0;

        if x == ToastDismissalReason::ApplicationHidden.0 {
          Self::ApplicationHidden
        } else if x == ToastDismissalReason::UserCanceled.0 {
          Self::UserCanceled
        } else if x == ToastDismissalReason::TimedOut.0 {
          Self::TimedOut
        } else {
          Self::Unknown(format!("Unknown reason: {x}"))
        }
      },
    )
  }
}

pub struct NotificationDismissedEventHandler {
  pub(crate) handler: TypedEventHandler<ToastNotification, ToastDismissedEventArgs>,
}

impl NotificationDismissedEventHandler {
  pub fn new<
    T: Fn(Option<PartialNotification>, Option<ToastDismissedReason>) -> Result<(), Error>
      + Send
      + Sync
      + 'static,
  >(
    func: T,
  ) -> Self {
    let handler: TypedEventHandler<ToastNotification, ToastDismissedEventArgs> =
      TypedEventHandler::new(
        move |a: Ref<ToastNotification>, b: Ref<ToastDismissedEventArgs>| {
          let a = a.as_ref();
          let a = a.and_then(|a| PartialNotification { _toast: a }.into());

          let b = b.as_ref();
          let b = b.and_then(|x| x.cast::<ToastDismissedEventArgs>().ok());
          let b = b.and_then(|x| Some(ToastDismissedReason::new(x)));

          func(a, b)
        },
      );

    Self { handler }
  }
}
