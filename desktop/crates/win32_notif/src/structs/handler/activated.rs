use std::collections::HashMap;

use windows::{
  core::{Error, IInspectable, Interface, Ref, HSTRING},
  Foundation::{IReference, TypedEventHandler},
  UI::Notifications::{ToastActivatedEventArgs, ToastNotification},
};

use crate::notification::PartialNotification;

#[derive(Debug)]
pub struct ToastActivatedArgs {
  pub button_id: Option<String>,
  pub user_input: Option<HashMap<String, String>>,
}

impl ToastActivatedArgs {
  pub(crate) fn new(args: ToastActivatedEventArgs) -> Self {
    let argument = args.Arguments().ok().and_then(|x| Some(x.to_string()));
    let user_input = args.UserInput().ok().and_then(|x| Some(x.into_iter()));

    let user_input = user_input.and_then(|x| {
      let mut val: HashMap<String, String> = HashMap::new();
      x.for_each(|x| {
        let _: Option<()> = (|| {
          let key = x.Key().ok()?;
          let key = key.to_string();
          let value = x.Value().ok()?;

          let value = value.cast::<IReference<HSTRING>>().ok();
          let value = value?.GetString().ok()?;

          let value = value.to_string();

          let _ = val.insert(key, value);
          Some(())
        })();
      });

      Some(val)
    });

    Self {
      button_id: argument,
      user_input,
    }
  }
}

pub struct NotificationActivatedEventHandler {
  pub(crate) handler: TypedEventHandler<ToastNotification, IInspectable>,
}

impl NotificationActivatedEventHandler {
  pub fn new<
    T: Fn(Option<PartialNotification>, Option<ToastActivatedArgs>) -> Result<(), Error>
      + Send
      + Sync
      + 'static,
  >(
    func: T,
  ) -> Self {
    let handler: TypedEventHandler<ToastNotification, IInspectable> =
      TypedEventHandler::new(move |a: Ref<ToastNotification>, b: Ref<IInspectable>| {
        let a = a.as_ref();
        let a = a.and_then(|a| PartialNotification { _toast: a }.into());

        let b = b.as_ref();
        let b = b.and_then(|x| x.cast::<ToastActivatedEventArgs>().ok());
        let b = b.and_then(|x| Some(ToastActivatedArgs::new(x)));

        func(a, b)
      });

    Self { handler }
  }
}
