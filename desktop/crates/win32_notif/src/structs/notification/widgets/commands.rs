use crate::ToXML;

/// Learn more about it here
/// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-commands>
pub struct Commands {
  widgets: Vec<Command>,
}

impl Commands {
  pub fn new(commands: Vec<Command>) -> Self {
    Self { widgets: commands }
  }
}

impl IntoIterator for Commands {
  type Item = Command;
  type IntoIter = std::vec::IntoIter<Self::Item>;

  fn into_iter(self) -> Self::IntoIter {
    self.widgets.into_iter()
  }
}

/// Learn more about it here
/// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-command>
pub struct Command {
  id: String,
  arguments: String,
}

impl Command {
  pub fn new(arguments: Option<String>, id: Option<CommandId>) -> Self {
    if let Some(x) = &arguments {
      debug_assert!(x.chars().all(|x| x.is_alphanumeric()));
    }

    Self {
      id: id.map_or_else(
        || "".into(),
        |x| format!("id=\"{}\"", Into::<String>::into(x)),
      ),
      arguments: arguments.map_or_else(|| "".into(), |x| format!("arguments=\"{}\"", x)),
    }
  }
}

pub enum CommandId {
  Snooze,
  Dismiss,
  Video,
  Voice,
  Decline,
}

impl Into<String> for CommandId {
  fn into(self) -> String {
    match self {
      Self::Snooze => "snooze".to_string(),
      Self::Dismiss => "dismiss".to_string(),
      Self::Video => "video".to_string(),
      Self::Voice => "voice".to_string(),
      Self::Decline => "decline".to_string(),
    }
  }
}

impl ToXML for Command {
  fn to_xml(&self) -> String {
    format!(
      r"
        <command {} {} />
      ",
      self.arguments, self.id
    )
  }
}
