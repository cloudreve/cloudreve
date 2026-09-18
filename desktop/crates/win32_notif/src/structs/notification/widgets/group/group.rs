use crate::{
  notification::{visual::VisualElement, ToastVisualableXML},
  ToXML,
};

use super::SubgroupXML;

/// Learn More Here
/// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-group>
pub struct Group {
  subgroups: Vec<Box<dyn SubgroupXML>>,
}

impl VisualElement for Group {}
impl ToastVisualableXML for Group {}

impl Group {
  pub fn new() -> Self {
    Self::default()
  }

  pub fn with_subgroup<T: SubgroupXML + 'static>(mut self, subgroup: T) -> Self {
    self.subgroups.push(Box::new(subgroup));
    self
  }

  pub fn new_from(subgroups: Vec<Box<dyn SubgroupXML>>) -> Self {
    Self { subgroups }
  }
}

impl Default for Group {
  fn default() -> Self {
    Self { subgroups: vec![] }
  }
}

impl ToXML for Group {
  fn to_xml(&self) -> String {
    let data = self
      .subgroups
      .iter()
      .map(|x| x.to_xml())
      .collect::<Vec<_>>()
      .join("\n");

    format!(
      "
      <group>
        {data}
      </group>
    "
    )
  }
}
