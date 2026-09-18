use std::collections::HashMap;

use crate::NotifError;

use super::{
  handler::{NotificationDismissedEventHandler, NotificationFailedEventHandler},
  NotificationActivatedEventHandler, NotificationImpl, ToXML, ToastsNotifier,
};
use actions::ActionElement;
use audio::Audio;
use header::Header;
use visual::VisualElement;
use widgets::commands::Commands;
use windows::{
  core::HSTRING,
  Data::Xml::Dom::XmlDocument,
  Foundation::{DateTime, IReference, PropertyValue},
  Globalization::Calendar,
  UI::Notifications::{NotificationData, ToastNotification},
};
use windows_core::Interface;

use std::time::Duration;

mod widgets;
pub use widgets::*;

/// This is a partial version of notification
/// You can convert it to a Notification **but it will lost the handler tokens**
///
/// We have to call [`OwnedPartialNotification::get_partial`] to get the PartialNotification object
/// to work on it
pub struct OwnedPartialNotification {
  pub(crate) notif: ToastNotification,
}

impl OwnedPartialNotification {
  pub fn get_partial<'a>(&'a self) -> PartialNotification<'a> {
    PartialNotification {
      _toast: &self.notif,
    }
  }
}

/// This is a partial version of notification
/// You can convert it to a Notification **but it will lost the handler tokens**
pub struct PartialNotification<'a> {
  pub(crate) _toast: &'a ToastNotification,
}

impl<'a> PartialNotification<'a> {
  #[deprecated = "Use `upgrade` instead"]
  pub fn cast(self, notifier: &'a ToastsNotifier) -> Notification<'a> {
    self.upgrade(notifier)
  }

  /// Converts to a Notification **but it will lost the handler tokens**
  pub fn upgrade(self, notifier: &'a ToastsNotifier) -> Notification<'a> {
    Notification {
      _toast: self._toast.clone(),
      _notifier: notifier,
      activated_event_handler_token: None,
      dismissed_event_handler_token: None,
      failed_event_handler_token: None,
    }
  }
}

impl<'a> NotificationImpl for PartialNotification<'a> {
  fn notif(&self) -> &ToastNotification {
    &self._toast
  }
}

impl<'a> Notification<'a> {
  pub fn show(&self) -> Result<(), NotifError> {
    Ok(self._notifier.get_raw_handle().Show(&self._toast)?)
  }

  pub unsafe fn as_raw(&self) -> &ToastNotification {
    &self._toast
  }
}

/// The Notification Object
pub struct Notification<'a> {
  pub(crate) _toast: ToastNotification,
  pub(crate) _notifier: &'a ToastsNotifier,
  pub activated_event_handler_token: Option<i64>,
  pub dismissed_event_handler_token: Option<i64>,
  pub failed_event_handler_token: Option<i64>,
}

impl NotificationImpl for Notification<'_> {
  fn notif(&self) -> &ToastNotification {
    &self._toast
  }
}

pub enum ToastDuration {
  None,
  Long,
  Short,
}

pub enum Scenario {
  Default,
  Reminder,
  Alarm,
  IncomingCall,
  Urgent,
}

pub trait ActionableXML: ActionElement + ToXML {}
pub trait ToastVisualableXML: VisualElement + ToXML {}

/// The way to build a Notification
pub struct NotificationBuilder {
  audio: Option<Audio>,
  header: Option<Header>,
  commands: Option<Commands>,
  expiry: Option<Duration>,
  visual: Vec<Box<dyn ToastVisualableXML>>,
  actions: Vec<Box<dyn ActionableXML>>,
  on_activated: Option<NotificationActivatedEventHandler>,
  on_failed: Option<NotificationFailedEventHandler>,
  on_dismissed: Option<NotificationDismissedEventHandler>,
  duration: &'static str,
  scenario: &'static str,
  use_button_style: &'static str,
  launch: String,  
  pub values: HashMap<String, String>,
}

macro_rules! impl_mut {
  ($x:ident -> $y:tt) => {
    pub fn $x(mut self, $x: $y) -> Self {
      self.$x = Some($x);
      self
    }
  };
}

#[macro_export]
#[doc(hidden)]
macro_rules! map {
  ($x:expr) => {
    $x.into_iter()
      .map(|x| x.to_xml())
      .collect::<Vec<_>>()
      .join("\n".into())
  };
}

impl NotificationBuilder {
  pub fn new() -> Self {
    Self {
      visual: vec![],
      actions: vec![],
      audio: None,
      commands: None,
      header: None,
      expiry: None,
      on_activated: None,
      on_dismissed: None,
      on_failed: None,
      duration: "",
      scenario: "",
      use_button_style: "",
      launch: String::new(),
      values: HashMap::new(),
    }
  }

  impl_mut!(audio -> Audio);
  impl_mut!(header -> Header);
  impl_mut!(commands -> Commands);

  pub fn with_duration(mut self, duration: ToastDuration) -> Self {
    match duration {
      ToastDuration::None => self.duration = "",
      ToastDuration::Short => self.duration = "duration=\"short\"",
      ToastDuration::Long => self.duration = "duration=\"long\"",
    }
    self
  }

  /// Sets the ExpirationTime of the notification
  ///
  /// Please note that its accurate upto **seconds only**
  ///
  /// ## Example
  /// ```rust
  /// fn main() {
  ///   let builder = NotificationBuilder::new()
  ///     .with_expiry(Duration::from_secs(30));
  /// }
  /// ```
  pub fn with_expiry(mut self, expiry: Duration) -> Self {
    self.expiry = Some(expiry);
    self
  }

  pub fn with_scenario(mut self, scenario: Scenario) -> Self {
    match scenario {
      Scenario::Default => self.scenario = "",
      Scenario::Alarm => self.scenario = "scenario=\"alarm\"",
      Scenario::Reminder => self.scenario = "scenario=\"reminder\"",
      Scenario::IncomingCall => self.scenario = "scenario=\"incomingCall\"",
      Scenario::Urgent => self.scenario = "scenario=\"urgent\"",
    }
    self
  }

  pub fn with_use_button_style(mut self, use_button_style: bool) -> Self {
    if use_button_style {
      self.use_button_style = "useButtonStyle=\"True\""
    } else {
      self.use_button_style = ""
    }
    self
  }

  pub fn value<T: Into<String>, E: Into<String>>(mut self, key: T, value: E) -> Self {
    self.values.insert(key.into(), value.into());
    self
  }

  pub fn with_launch<T: Into<String>>(mut self, launch: T) -> Self {
    self.launch = format!("launch=\"{}\"", launch.into());
    self
  }

  pub fn values(mut self, values: HashMap<String, String>) -> Self {
    self.values = values;
    self
  }

  pub fn action<T: ActionableXML + 'static>(mut self, action: T) -> Self {
    self.actions.push(Box::new(action));
    self
  }

  pub fn actions(mut self, actions: Vec<Box<dyn ActionableXML>>) -> Self {
    self.actions = actions;
    self
  }

  pub fn visual<T: ToastVisualableXML + 'static>(mut self, visual: T) -> Self {
    self.visual.push(Box::new(visual));
    self
  }

  pub fn visuals(mut self, visual: Vec<Box<dyn ToastVisualableXML>>) -> Self {
    self.visual = visual;
    self
  }

  pub fn on_activated(mut self, on_activated: NotificationActivatedEventHandler) -> Self {
    self.on_activated = Some(on_activated);
    self
  }

  pub fn on_failed(mut self, on_failed: NotificationFailedEventHandler) -> Self {
    self.on_failed = Some(on_failed);
    self
  }

  pub fn on_dismissed(mut self, on_dismissed: NotificationDismissedEventHandler) -> Self {
    self.on_dismissed = Some(on_dismissed);
    self
  }

  pub fn build<'a>(
    self,
    sequence: u32,
    _notifier: &'a ToastsNotifier,
    tag: &str,
    group: &str,
  ) -> Result<Notification<'a>, NotifError> {
    let visual = map!(self.visual);
    let actions = map!(self.actions);

    let audio = self.audio.map_or_else(|| "".into(), |x| x.to_xml());
    let header = self.header.map_or_else(|| "".into(), |x| x.to_xml());

    let commands = self.commands.map_or_else(
      || "".into(),
      |x| {
        format!(
          r"
        <commands>
          {}
        </commands>
      ",
          map!(x)
        )
      },
    );

    let _xml = format!(
      r#"
      <toast {dur} {scenario} {button_style} {launch}>
        {audio}
        {commands}
        {header}
        <visual>
          <binding template='ToastGeneric'>
            {visual}
          </binding>
        </visual>
        <actions>
          {actions}
        </actions>
      </toast>
    "#,
      dur = self.duration,
      scenario = self.scenario,
      button_style = self.use_button_style,
      launch = self.launch
    );

    let doc = XmlDocument::new()?;
    doc.LoadXml(&HSTRING::from(_xml))?;

    let data = NotificationData::new()?;
    data.SetSequenceNumber(sequence)?;

    for (key, value) in self.values {
      data.Values()?.Insert(&key.into(), &value.into())?;
    }

    let mut activated_event_handler_token = None;
    let mut dismissed_event_handler_token = None;
    let mut failed_event_handler_token = None;

    let toast = ToastNotification::CreateToastNotification(&doc)?;
    if let Some(x) = self.on_activated {
      let token = toast.Activated(&x.handler)?;
      activated_event_handler_token = Some(token);
    }
    if let Some(x) = self.on_dismissed {
      let token = toast.Dismissed(&x.handler)?;
      dismissed_event_handler_token = Some(token);
    }
    if let Some(x) = self.on_failed {
      let token = toast.Failed(&x.handler)?;
      failed_event_handler_token = Some(token);
    }

    if let Some(x) = self.expiry {
      let calendar = Calendar::new()?;

      if x.as_secs() > i32::MAX as u64 {
        return Err(NotifError::DurationTooLong);
      }

      calendar.AddSeconds(x.as_secs() as i32)?;

      let dt = calendar.GetDateTime()?;

      toast
        .SetExpirationTime(&PropertyValue::CreateDateTime(dt)?.cast::<IReference<DateTime>>()?)?;
    }

    toast.SetTag(&tag.into())?;
    toast.SetGroup(&group.into())?;
    toast.SetData(&data)?;

    Ok(Notification {
      _toast: toast,
      _notifier,
      activated_event_handler_token,
      dismissed_event_handler_token,
      failed_event_handler_token,
    })
  }
}
