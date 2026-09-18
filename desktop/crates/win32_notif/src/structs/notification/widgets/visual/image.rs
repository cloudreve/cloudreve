use quick_xml::escape::escape;

use crate::{notification::ToastVisualableXML, ToXML};

use super::{TextOrImageElement, VisualElement};

/// Learn more here
/// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-image#attributes>
pub enum Placement {
  AppLogoOverride,
  Hero,
  None,
}

impl ToString for Placement {
  fn to_string(&self) -> String {
    match self {
      Placement::AppLogoOverride => "placement=\"appLogoOverride\"".to_string(),
      Placement::Hero => "placement=\"hero\"".to_string(),
      Placement::None => "".to_string(),
    }
  }
}

#[derive(Debug, Clone, Default)]
pub enum ImageCrop {
  #[default]
  Default,
  None,
  Circle,
}

impl ToString for ImageCrop {
  fn to_string(&self) -> String {
    match self {
      ImageCrop::Default => "".to_string(),
      ImageCrop::Circle => "hint-crop=\"circle\"".to_string(),
      ImageCrop::None => "hint-crop=\"none\"".to_string(),
    }
  }
}

#[derive(Debug, Clone, Default)]
pub enum AdaptiveImageAlign {
  #[default]
  Default,
  Stretch,
  Left,
  Center,
  Right,
}

impl ToString for AdaptiveImageAlign {
  fn to_string(&self) -> String {
    match self {
      AdaptiveImageAlign::Default => "".to_string(),
      AdaptiveImageAlign::Stretch => "hint-align=\"stretch\"".to_string(),
      AdaptiveImageAlign::Left => "hint-align=\"left\"".to_string(),
      AdaptiveImageAlign::Center => "hint-align=\"center\"".to_string(),
      AdaptiveImageAlign::Right => "hint-align=\"right\"".to_string(),
    }
  }
}

#[allow(non_snake_case)]
/// Learn more here
/// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-image>
pub struct Image {
  pub id: u64,
  pub src: String,
  pub alt: Option<String>,
  pub add_image_query: bool,
  pub placement: Placement,
  pub crop: ImageCrop,
  pub no_margin: bool,
  pub align: AdaptiveImageAlign,
}

impl TextOrImageElement for Image {}

fn guess_src(src: String) -> String {
  let protocols = [
    "https://",
    "http://",
    "file:///",
    "ms-appx:///",
    "ms-appdata:///local/",
  ];

  if !(protocols.iter().any(|x| src.starts_with(x))) {
    return format!("file:///{src}");
  }

  src
}

impl Image {
  /// The `src` should be the either of the following following
  /// - `https://url or http://url`
  /// - `file:///path/to/file`
  ///
  /// If none of the above is provided, the `src` will be set to `file:///path/to/file`
  pub fn create(id: u64, src: &str) -> Self {
    Self::new(
      id,
      escape(src).into(),
      None,
      false,
      Placement::None,
      ImageCrop::Default,
      false,
    )
  }

  /// The `src` should be in the form of `file:///path/to/file`
  /// 
  /// Technically `https://` and `http://` too should work according to the
  /// C# Windows UWP Notifications, but we were not able to replicate that.
  /// 
  /// We still allow setting http or https including others of windows C# API
  ///
  /// If none of the above is provided, the `src` will be set to `file:///path/to/file`
  pub fn new(
    id: u64,
    src: String,
    alt: Option<String>,
    add_image_query: bool,
    placement: Placement,
    crop: ImageCrop,
    no_margin: bool,
  ) -> Self {
    Self {
      id,
      add_image_query,
      src: guess_src(src),
      alt,
      placement,
      crop,
      align: AdaptiveImageAlign::Default,
      no_margin,
    }
  }
}

impl Image {
  pub fn with_margin(mut self, margin: bool) -> Self {
    self.no_margin = !margin;
    self
  }

  pub fn with_align(mut self, align: AdaptiveImageAlign) -> Self {
    self.align = align;
    self
  }

  pub fn with_alt<S: Into<String>>(mut self, alt: S) -> Self {
    self.alt = Some(alt.into());
    self
  }

  pub fn without_image_query(mut self) -> Self {
    self.add_image_query = false;
    self
  }

  pub fn with_image_query(mut self) -> Self {
    self.add_image_query = true;
    self
  }

  pub fn with_crop(mut self, crop: ImageCrop) -> Self {
    self.crop = crop;
    self
  }

  pub fn with_placement(mut self, placement: Placement) -> Self {
    self.placement = placement;
    self
  }
}

impl VisualElement for Image {}

impl ToastVisualableXML for Image {}

impl ToXML for Image {
  fn to_xml(&self) -> String {
    format!(
      r#"
        <image id="{id}" {margin} {align} src="{src}" {add_image_query} {alt} {placement} {crop} />
      "#,
      align = self.align.to_string(),
      margin = match self.no_margin {
        true => "hint-remove-margin=\"true\"".to_string(),
        false => "".to_string(),
      },
      id = self.id,
      src = format!("{}", self.src).replace("\\\\", "\\"),
      alt = self
        .alt
        .clone()
        .map_or_else(|| string!(""), |x| format!("alt=\"{x}\"")),
      add_image_query = if self.add_image_query {
        "addImageQuery=\"True\""
      } else {
        ""
      },
      placement = self.placement.to_string(),
      crop = self.crop.to_string()
    )
  }
}
