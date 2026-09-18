use crate::{notification::visual::TextOrImageElement, ToXML};

use super::SubgroupXML;

/// Learn More Here
/// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-subgroup>
pub struct SubGroup {
  elements: Vec<Box<dyn TextOrImageElement>>,
  weight: u16,
  text_stacking: Option<&'static str>
}

pub enum AdaptiveSubgroupTextStacking {
  Default,
  Top,
  Center,
  Bottom
}

impl AdaptiveSubgroupTextStacking {
  pub fn to_string(&self) -> &'static str {
    match self {
      AdaptiveSubgroupTextStacking::Default => "default",
      AdaptiveSubgroupTextStacking::Top => "top",
      AdaptiveSubgroupTextStacking::Center => "center",
      AdaptiveSubgroupTextStacking::Bottom => "bottom",
    }
  }
}

impl SubgroupXML for SubGroup {}

impl SubGroup {
  pub fn new() -> Self {
    Self::default()
  }

  pub fn with_visual<T: TextOrImageElement + 'static>(mut self, element: T) -> Self {
    self.elements.push(Box::new(element));
    self
  }

  pub fn with_weight(mut self, weight: u16) -> Self {
    self.weight = weight;
    self
  }

  pub fn with_text_stacking(mut self, stack: Option<AdaptiveSubgroupTextStacking>) -> Self {
    self.text_stacking = stack.map(|x| x.to_string());
    self
  }

  pub fn new_from(elements: Vec<Box<dyn TextOrImageElement>>) -> Self {
    Self { elements, ..Default::default() }
  }
}

impl Default for SubGroup {
  fn default() -> Self {
    Self { elements: vec![], text_stacking: None, weight: 0 }
  }
}

impl ToXML for SubGroup {
  fn to_xml(&self) -> String {
    let data = self
      .elements
      .iter()
      .map(|x| x.to_xml())
      .collect::<Vec<_>>()
      .join("\n");

    // XML Formatting
    format!(
      "
      <subgroup {wt} {stack}>
        {data}
      </subgroup>
    ",
      wt = if self.weight == 0 {
        "".to_string()
      } else {
        format!("hint-weight=\"{}\"", self.weight)
      },
      stack = self.text_stacking.map_or_else(
        || String::new(),
        |s| format!("hint-textStacking=\"{}\"", s)
      )
    )
  }
}
