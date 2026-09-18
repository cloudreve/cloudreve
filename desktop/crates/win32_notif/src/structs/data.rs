use windows::UI::Notifications::NotificationData;

use crate::NotifError;

pub struct NotificationDataSet {
  _inner: NotificationData,
}

impl NotificationDataSet {
  pub fn new() -> Result<Self, NotifError> {
    Ok(Self {
      _inner: NotificationData::new()?,
    })
  }

  pub fn insert(&self, k: &str, v: &str) -> Result<bool, NotifError> {
    Ok(self._inner.Values()?.Insert(&k.into(), &v.into())?)
  }

  pub fn inner_win32_type(&self) -> &NotificationData {
    &self._inner
  }
}
