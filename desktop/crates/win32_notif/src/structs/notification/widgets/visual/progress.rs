use crate::{
  notification::{AdaptiveText, ToastVisualableXML},
  ToXML,
};

use super::VisualElement;

#[allow(non_snake_case)]
/// Learn more here
/// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-progress>
pub struct Progress {
  title: Option<String>,
  value_string_override: Option<String>,
  status: String,
  value: String,
}

pub enum ProgressValue<'a> {
  Percentage(f64),
  BindTo(&'a str),
  Indeterminate,
}

impl<'a> ToString for ProgressValue<'a> {
  fn to_string(&self) -> String {
    match self {
      ProgressValue::Percentage(x) => format!("{}", x / 100.0),
      ProgressValue::BindTo(x) => {
        debug_assert!(x.chars().all(|x| x.is_alphabetic()));

        format!("{{{x}}}")
      }
      ProgressValue::Indeterminate => "indeterminate".to_string(),
    }
  }
}

impl Progress {
  pub fn create(status_text: AdaptiveText, value: ProgressValue) -> Self {
    unsafe { Self::new_unchecked(None, status_text.to_string(), value, None) }
  }

  pub fn with_title<T: AsRef<str>>(mut self, title: AdaptiveText) -> Self {
    self.title = Some(title.to_string());
    self
  }

  pub fn with_value(mut self, value: ProgressValue) -> Self {
    self.value = value.to_string();
    self
  }

  pub fn with_override_value(mut self, value: AdaptiveText) -> Self {
    self.value_string_override = Some(value.to_string());
    self
  }

  pub unsafe fn new_unchecked(
    title: Option<String>,
    status_text: String,
    value: ProgressValue,
    value_string_override: Option<String>,
  ) -> Self {
    Self {
      title,
      status: status_text,
      value: value.to_string(),
      value_string_override,
    }
  }
}

impl VisualElement for Progress {}

impl ToastVisualableXML for Progress {}

impl ToXML for Progress {
  fn to_xml(&self) -> String {
    format!(
      r#"
        <progress {} status="{}" value="{}" {} />
      "#,
      self
        .title
        .clone()
        .map_or_else(|| string!(""), |x| format!("title=\"{x}\"")),
      self.status,
      self.value,
      self
        .value_string_override
        .clone()
        .map_or_else(|| string!(""), |x| format!("valueStringOverride=\"{x}\""))
    )
  }
}
