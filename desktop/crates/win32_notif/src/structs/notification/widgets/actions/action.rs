use quick_xml::escape::escape;

use crate::{notification::ActionableXML, ToXML};

use super::ActionElement;

#[allow(non_snake_case)]
/// Learn More Here
/// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-action>
pub struct ActionButton {
  content: String,
  arguments: String,
  imageUri: Option<String>,
  hint_inputid: String,
  hint_toolTip: String,

  activationType: String,
  afterActivationBehavior: String,
  hint_buttonStyle: String,
  placement: bool,
}

#[allow(non_snake_case)]
impl ActionButton {
  pub fn create<T: AsRef<str>>(content: T) -> Self {
    unsafe {
      Self::new_unchecked(
        escape(content.as_ref()).into(),
        escape(content.as_ref()).into(),
        ActivationType::Foreground,
        AfterActivationBehavior::Default,
        None,
        "".into(),
        HintButtonStyle::None,
        "".into(),
        false,
      )
    }
  }

  pub fn with_id(mut self, id: &str) -> Self {
    self.arguments = escape(id).into();
    self
  }

  /// Provide input id to place the button near an input
  pub fn with_input_id(mut self, id: &str) -> Self {
    self.hint_inputid = escape(id).into();
    self
  }

  pub fn with_tooltip(mut self, tooltip: &str) -> Self {
    self.hint_toolTip = escape(tooltip).into();
    self
  }

  pub fn with_image_uri(mut self, uri: &str) -> Self {
    self.imageUri = Some(escape(uri).into());
    self
  }

  pub fn with_context_menu_placement(mut self, enabled: bool) -> Self {
    self.placement = enabled;
    self
  }

  pub fn with_activation_type(mut self, activation_type: ActivationType) -> Self {
    self.activationType = activation_type.into();
    self
  }

  pub fn with_after_activation_behavior(
    mut self,
    after_activation_behavior: AfterActivationBehavior,
  ) -> Self {
    self.afterActivationBehavior = after_activation_behavior.into();
    self
  }

  pub fn with_button_style(mut self, hint_buttonStyle: HintButtonStyle) -> Self {
    self.hint_buttonStyle = hint_buttonStyle.into();
    self
  }

  pub fn with_content(mut self, content: &str) -> Self {
    self.content = escape(content).into();
    self
  }

  pub unsafe fn new_unchecked(
    content: String,
    arguments: String,
    activation_type: ActivationType,
    after_activation_behavior: AfterActivationBehavior,
    image_uri: Option<String>,
    hint_inputid: String,
    hint_buttonStyle: HintButtonStyle,
    hint_toolTip: String,
    placement: bool,
  ) -> Self {
    Self {
      content,
      arguments,
      activationType: activation_type.into(),
      afterActivationBehavior: after_activation_behavior.into(),
      imageUri: image_uri,
      hint_inputid,
      hint_buttonStyle: hint_buttonStyle.into(),
      hint_toolTip,
      placement,
    }
  }
}

impl ToXML for ActionButton {
  fn to_xml(&self) -> String {
    format!(
      r#"
          <action content="{}" arguments="{}" activationType="{}" afterActivationBehavior="{}" imageUri="{}" hint-inputId="{}" hint-buttonStyle="{}" hint-toolTip="{}" {} />
        "#,
      self.content,
      self.arguments,
      self.activationType,
      self.afterActivationBehavior,
      self.imageUri.as_ref().unwrap_or(&"".to_string()),
      self.hint_inputid,
      self.hint_buttonStyle,
      self.hint_toolTip,
      if self.placement {
        "placement=\"contextMenu\""
      } else {
        ""
      }
    )
  }
}

#[derive(Default)]
/// Learn More Here
/// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-action>
pub enum ActivationType {
  #[default]
  Foreground,
  Background,
  Protocol,
}

impl Into<String> for ActivationType {
  fn into(self) -> String {
    match self {
      ActivationType::Foreground => "foreground".to_string(),
      ActivationType::Background => "background".to_string(),
      ActivationType::Protocol => "protocol".to_string(),
    }
  }
}

#[derive(Default)]
/// Learn More Here
/// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-action>
pub enum AfterActivationBehavior {
  #[default]
  Default,
  PendingUpdate,
}

impl Into<String> for AfterActivationBehavior {
  fn into(self) -> String {
    match self {
      Self::Default => "default".to_string(),
      Self::PendingUpdate => "pendingUpdate".to_string(),
    }
  }
}

#[derive(Default)]
/// Learn More Here
/// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-action>
pub enum HintButtonStyle {
  #[default]
  None,
  Success,
  Critical,
}

impl Into<String> for HintButtonStyle {
  fn into(self) -> String {
    match self {
      Self::None => "".to_string(),
      Self::Success => "Success".to_string(),
      Self::Critical => "Critical".to_string(),
    }
  }
}

impl ActionElement for ActionButton {}
impl ActionableXML for ActionButton {}
