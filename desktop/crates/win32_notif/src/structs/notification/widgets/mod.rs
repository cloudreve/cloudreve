use quick_xml::escape::escape;

pub mod actions;
pub mod audio;
pub mod commands;
pub mod group;
pub mod header;
pub mod visual;

pub enum AdaptiveText<'a> {
  BindTo(&'a str),
  Text(&'a str),
}

impl<'a> From<&'a str> for AdaptiveText<'a> {
  fn from(value: &'a str) -> Self {
    Self::Text(value)
  }
}

impl<'a> From<&'a String> for AdaptiveText<'a> {
  fn from(value: &'a String) -> Self {
    Self::Text(value)
  }
}

impl<'a> ToString for AdaptiveText<'a> {
  fn to_string(&self) -> String {
    match self {
      AdaptiveText::Text(x) => escape(*x).to_string(),
      AdaptiveText::BindTo(x) => {
        debug_assert!(x.chars().all(|x| x.is_alphabetic()));

        format!("{{{x}}}")
      }
    }
  }
}

#[cfg_attr(docsrs, doc(cfg(feature = "experimental")))]
#[cfg(feature = "experimental")]
pub mod raw_xml;
