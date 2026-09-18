pub mod data;
pub mod handler;
pub mod notification;
pub mod notifier;

use std::time::Duration;

pub use data::NotificationDataSet;
pub use handler::{
  NotificationActivatedEventHandler, NotificationDismissedEventHandler,
  NotificationFailedEventHandler,
};

pub use notification::{Notification, NotificationBuilder};
pub use notifier::ToastsNotifier;
use windows::{
  core::HSTRING,
  Foundation::{DateTime, IReference, PropertyValue},
  Globalization::Calendar,
  UI::Notifications::{
    NotificationMirroring as ToastNotificationMirroring, ToastNotification,
    ToastNotificationPriority,
  },
};
use windows_core::Interface;

use crate::NotifError;

pub enum NotificationPriority {
  Default,
  High,
}

pub enum NotificationMirroring {
  Allowed,
  Disallowed,
}

#[cfg(not(feature = "unsafe"))]
pub(crate) trait ToXML {
  fn to_xml(&self) -> String;
}

#[cfg(feature = "unsafe")]
pub trait ToXML {
  fn to_xml(&self) -> String;
}

pub trait NotificationImpl {
  fn notif(&self) -> &ToastNotification;
}

macro_rules! implement {
  (get $x:ident $y:ident HSTRING) => {
    fn $x(&self) -> Result<String, NotifError> {
      Ok(self.notif().$y()?.to_string())
    }
  };

  (set $x:ident $y:ident String) => {
    fn $x(&self, data: String) -> Result<(), NotifError> {
      Ok(self.notif().$y(&HSTRING::from(data))?)
    }
  };

  (get $x:ident $y:ident bool) => {
    fn $x(&self) -> Result<bool, NotifError> {
      Ok(self.notif().$y()?)
    }
  };

  (set $x:ident $y:ident bool) => {
    fn $x(&self, data: bool) -> Result<(), NotifError> {
      Ok(self.notif().$y(data)?)
    }
  };
}

pub trait ManageNotification {
  fn get_xml_content(&self) -> Option<String>;

  fn priority(&self) -> Result<NotificationPriority, NotifError>;
  fn set_priority(&self, priority: NotificationPriority) -> Result<(), NotifError>;

  fn notification_mirroring(&self) -> Result<NotificationMirroring, NotifError>;
  fn set_notification_mirroring(&self, mirroring: NotificationMirroring) -> Result<(), NotifError>;

  fn set_expiration(&self, expires: Duration) -> Result<(), NotifError>;

  fn set_activated_handler(
    &self,
    handler: NotificationActivatedEventHandler,
  ) -> Result<i64, NotifError>;
  fn remove_activated_handler(&self, token: i64) -> Result<(), NotifError>;

  fn set_dismissed_handler(
    &self,
    handler: NotificationDismissedEventHandler,
  ) -> Result<i64, NotifError>;
  fn remove_dismissed_handler(&self, token: i64) -> Result<(), NotifError>;

  fn set_failed_handler(&self, handler: NotificationFailedEventHandler) -> Result<i64, NotifError>;
  fn remove_failed_handler(&self, token: i64) -> Result<(), NotifError>;

  fn get_tag(&self) -> Result<String, NotifError>;
  fn set_tag(&self, tag: String) -> Result<(), NotifError>;

  fn get_group(&self) -> Result<String, NotifError>;
  fn set_group(&self, group: String) -> Result<(), NotifError>;

  fn get_remote_id(&self) -> Result<String, NotifError>;
  fn set_remote_id(&self, remote: String) -> Result<(), NotifError>;

  fn suppress_popup(&self) -> Result<bool, NotifError>;
  fn set_suppress_popup(&self, value: bool) -> Result<(), NotifError>;

  fn expires_on_reboot(&self) -> Result<bool, NotifError>;
  fn set_expires_on_reboot(&self, value: bool) -> Result<(), NotifError>;
}

impl<T: NotificationImpl> ManageNotification for T {
  fn get_xml_content(&self) -> Option<String> {
    Some(self.notif().Content().ok()?.GetXml().ok()?.to_string())
  }

  fn set_expiration(&self, expires: Duration) -> Result<(), NotifError> {
    let calendar = Calendar::new()?;

    if expires.as_secs() > i32::MAX as u64 {
      return Err(NotifError::DurationTooLong);
    }

    calendar.AddSeconds(expires.as_secs() as i32)?;

    let dt = calendar.GetDateTime()?;

    self
      .notif()
      .SetExpirationTime(&PropertyValue::CreateDateTime(dt)?.cast::<IReference<DateTime>>()?)?;

    Ok(())
  }

  fn priority(&self) -> Result<NotificationPriority, NotifError> {
    let priority = self.notif().Priority()?.0;
    let def = ToastNotificationPriority::Default.0;
    let high = ToastNotificationPriority::High.0;

    if priority == def {
      return Ok(NotificationPriority::Default);
    } else if priority == high {
      return Ok(NotificationPriority::High);
    }

    Err(NotifError::UnknownAndImpossible)
  }
  fn set_priority(&self, priority: NotificationPriority) -> Result<(), NotifError> {
    let priority = match priority {
      NotificationPriority::Default => ToastNotificationPriority::Default,
      NotificationPriority::High => ToastNotificationPriority::High,
    };

    Ok(self.notif().SetPriority(priority)?)
  }

  fn notification_mirroring(&self) -> Result<NotificationMirroring, NotifError> {
    let mirroring = self.notif().NotificationMirroring()?.0;

    if mirroring == ToastNotificationMirroring::Allowed.0 {
      return Ok(NotificationMirroring::Allowed);
    } else if mirroring == ToastNotificationMirroring::Disabled.0 {
      return Ok(NotificationMirroring::Disallowed);
    }

    Err(NotifError::UnknownAndImpossible)
  }

  fn set_notification_mirroring(&self, mirroring: NotificationMirroring) -> Result<(), NotifError> {
    let mirroring = match mirroring {
      NotificationMirroring::Allowed => ToastNotificationMirroring::Allowed,
      NotificationMirroring::Disallowed => ToastNotificationMirroring::Disabled,
    };

    Ok(self.notif().SetNotificationMirroring(mirroring)?)
  }

  fn set_activated_handler(
    &self,
    handler: NotificationActivatedEventHandler,
  ) -> Result<i64, NotifError> {
    Ok(self.notif().Activated(&handler.handler)?)
  }
  fn remove_activated_handler(&self, token: i64) -> Result<(), NotifError> {
    Ok(self.notif().RemoveActivated(token)?)
  }

  fn set_dismissed_handler(
    &self,
    handler: NotificationDismissedEventHandler,
  ) -> Result<i64, NotifError> {
    Ok(self.notif().Dismissed(&handler.handler)?)
  }
  fn remove_dismissed_handler(&self, token: i64) -> Result<(), NotifError> {
    Ok(self.notif().RemoveDismissed(token)?)
  }

  fn set_failed_handler(&self, handler: NotificationFailedEventHandler) -> Result<i64, NotifError> {
    Ok(self.notif().Failed(&handler.handler)?)
  }
  fn remove_failed_handler(&self, token: i64) -> Result<(), NotifError> {
    Ok(self.notif().RemoveFailed(token)?)
  }

  implement! {
    get get_tag Tag HSTRING
  }
  implement! {
    set set_tag SetTag String
  }

  implement! {
    get get_group Group HSTRING
  }
  implement! {
    set set_group SetGroup String
  }

  implement! {
    get get_remote_id RemoteId HSTRING
  }
  implement! {
    set set_remote_id SetRemoteId String
  }

  implement! {
    get expires_on_reboot ExpiresOnReboot bool
  }
  implement! {
    set set_expires_on_reboot SetExpiresOnReboot bool
  }

  implement! {
    get suppress_popup SuppressPopup bool
  }
  implement! {
    set set_suppress_popup SetSuppressPopup bool
  }
}
