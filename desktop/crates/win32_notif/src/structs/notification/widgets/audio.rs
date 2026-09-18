use crate::ToXML;

/// Learn More About this here
/// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-audio>
pub struct Audio {
  src: String,
  r#loop: String,
  silent: String,
}

impl Audio {
  pub fn new(src: Src, r#loop: bool, silent: bool) -> Self {
    Self {
      src: src.into(),
      r#loop: r#loop.to_string(),
      silent: silent.to_string(),
    }
  }
}

impl ToXML for Audio {
  fn to_xml(&self) -> String {
    format!(
      r#"
        <audio src="{}" loop="{}" silent="{}" />
      "#,
      self.src, self.r#loop, self.silent
    )
  }
}

#[derive(Default)]
/// Learn More About it here
/// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-audio>
pub enum Src {
  #[default]
  Default,
  IM,
  Mail,
  Reminder,
  Sms,
  Alarm,
  Alarm2,
  Alarm3,
  Alarm4,
  Alarm5,
  Alarm6,
  Alarm7,
  Alarm8,
  Alarm9,
  Alarm10,
  Call,
  Call2,
  Call3,
  Call4,
  Call5,
  Call6,
  Call7,
  Call8,
  Call9,
  Call10,
}

impl Into<String> for Src {
  fn into(self) -> String {
    match self {
      Self::Default => "ms-winsoundevent:Notification.Default",
      Self::IM => "ms-winsoundevent:Notification.IM",
      Self::Mail => "ms-winsoundevent:Notification.Mail",
      Self::Reminder => "ms-winsoundevent:Notification.Reminder",
      Self::Sms => "ms-winsoundevent:Notification.Sms",
      Self::Alarm => "ms-winsoundevent:Notification.Looping.Alarm",
      Self::Alarm2 => "ms-winsoundevent:Notification.Looping.Alarm2",
      Self::Alarm3 => "ms-winsoundevent:Notification.Looping.Alarm3",
      Self::Alarm4 => "ms-winsoundevent:Notification.Looping.Alarm4",
      Self::Alarm5 => "ms-winsoundevent:Notification.Looping.Alarm5",
      Self::Alarm6 => "ms-winsoundevent:Notification.Looping.Alarm6",
      Self::Alarm7 => "ms-winsoundevent:Notification.Looping.Alarm7",
      Self::Alarm8 => "ms-winsoundevent:Notification.Looping.Alarm8",
      Self::Alarm9 => "ms-winsoundevent:Notification.Looping.Alarm9",
      Self::Alarm10 => "ms-winsoundevent:Notification.Looping.Alarm10",
      Self::Call => "ms-winsoundevent:Notification.Looping.Call",
      Self::Call2 => "ms-winsoundevent:Notification.Looping.Call2",
      Self::Call3 => "ms-winsoundevent:Notification.Looping.Call3",
      Self::Call4 => "ms-winsoundevent:Notification.Looping.Call4",
      Self::Call5 => "ms-winsoundevent:Notification.Looping.Call5",
      Self::Call6 => "ms-winsoundevent:Notification.Looping.Call6",
      Self::Call7 => "ms-winsoundevent:Notification.Looping.Call7",
      Self::Call8 => "ms-winsoundevent:Notification.Looping.Call8",
      Self::Call9 => "ms-winsoundevent:Notification.Looping.Call9",
      Self::Call10 => "ms-winsoundevent:Notification.Looping.Call10",
    }
    .into()
  }
}
