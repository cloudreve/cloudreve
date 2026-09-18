use quick_xml::escape::escape;

use crate::{map, notification::ActionableXML, ToXML};

use super::ActionElement;

#[allow(non_snake_case)]
/// Learn more here
/// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-input>
pub struct Input {
  id: String,
  title: String,
  placeHolder: String,
  children: String,
  r#type: String,
  defaultInput: String,
}

/// Learn more here
/// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-input>
pub enum InputType {
  Text,
  Selection(Vec<Selection>),
}

impl Input {
  pub fn create_text_input(id: &str, title: &str, place_holder: &str) -> Self {
    unsafe {
      Self::new_unchecked(
        escape(id).into(),
        escape(title).into(),
        InputType::Text,
        escape(place_holder).into(),
      )
    }
  }

  pub fn create_selection_input(
    id: &str,
    title: &str,
    place_holder: &str,
    selections: Vec<Selection>,
    default_input: &str,
  ) -> Self {
    Self {
      id: escape(id).into(),
      title: escape(title).into(),
      r#type: "selection".into(),
      placeHolder: escape(place_holder).into(),
      children: map!(selections),
      defaultInput: default_input.into(),
    }
  }

  pub unsafe fn new_unchecked(
    id: String,
    title: String,
    r#type: InputType,
    place_holder: String,
  ) -> Self {
    let (r#type, ch) = match r#type {
      InputType::Text => ("text", vec![]),
      InputType::Selection(ch) => ("selection", ch),
    };

    Self {
      children: map!(ch),
      id,
      title,
      r#type: r#type.into(),
      placeHolder: place_holder,
      defaultInput: String::new(),
    }
  }

  pub fn with_selection(&mut self, children: Vec<Selection>) -> &mut Self {
    self.children = map!(children);
    self
  }
}

impl ActionElement for Input {}

impl ToXML for Input {
  fn to_xml(&self) -> String {
    format!(
      r#"
        <input id="{}" title="{}" placeHolderContent="{}" type="{}" defaultInput="{}" >
          {}
        </input>
      "#,
      self.id, self.title, self.placeHolder, self.r#type, self.defaultInput, self.children
    )
  }
}

/// Learn more here
/// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-input>
pub struct Selection {
  id: String,
  content: String,
}

impl Selection {
  pub fn new(id: &str, content: &str) -> Self {
    Self {
      content: escape(content).into(),
      id: escape(id).into(),
    }
  }
}

impl ToXML for Selection {
  fn to_xml(&self) -> String {
    format!(
      r#"<selection id="{}" content="{}" />"#,
      &self.id, &self.content
    )
  }
}

impl ActionableXML for Input {}
