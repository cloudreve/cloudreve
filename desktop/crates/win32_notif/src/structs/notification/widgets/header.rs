use quick_xml::escape::escape;

use crate::ToXML;

/// Learn more about it here
/// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-header>
pub struct Header {
  id: String,
  title: String,
  arguments: String,
  activation_type: String,
}

impl Header {
  pub fn new(
    id: &str,
    title: &str,
    arguments: &str,
    activation_type: Option<HeaderActivationType>,
  ) -> Self {
    Self {
      id: escape(id).into(),
      title: escape(title).into(),
      arguments: escape(arguments).into(),
      activation_type: activation_type.unwrap_or_default().into(),
    }
  }
}

impl ToXML for Header {
  fn to_xml(&self) -> String {
    format!(
      r#"
      <header title="{}" arguments="{}" id="{}" activationType="{}" />
    "#,
      self.title, self.arguments, self.id, self.activation_type
    )
  }
}

#[derive(Default)]
/// Learn more about it here
/// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-header>
pub enum HeaderActivationType {
  #[default]
  Foreground,
  Protocol,
}

impl Into<String> for HeaderActivationType {
  fn into(self) -> String {
    match self {
      HeaderActivationType::Foreground => "foreground".to_string(),
      HeaderActivationType::Protocol => "protocol".to_string(),
    }
  }
}
