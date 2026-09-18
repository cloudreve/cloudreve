use crate::notification::actions::ActionElement;
use crate::notification::group::SubgroupXML;
use crate::notification::visual::VisualElement;

use crate::ToXML;

use crate::notification::{ActionableXML, ToastVisualableXML};

use super::visual::TextOrImageElement;

pub struct RawXML {
  raw: String,
}

impl RawXML {
  /// Creates a new instance of `RawXML` that can hold arbitrary String
  /// This is useful when you want to use a widget that is not yet supported
  ///
  /// # Safety
  /// This function is unsafe because it bypasses all the safety that other structs guarantee
  pub unsafe fn new<T: ToString>(raw: T) -> Self {
    Self {
      raw: raw.to_string(),
    }
  }
}

impl ActionElement for RawXML {}

impl ActionableXML for RawXML {}

impl VisualElement for RawXML {}

impl ToastVisualableXML for RawXML {}

impl SubgroupXML for RawXML {}

impl TextOrImageElement for RawXML {}

impl ToXML for RawXML {
  fn to_xml(&self) -> String {
    self.raw.clone()
  }
}
