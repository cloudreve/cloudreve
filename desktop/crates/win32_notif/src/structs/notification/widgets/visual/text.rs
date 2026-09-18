use quick_xml::escape::escape;

use crate::{notification::ToastVisualableXML, ToXML};

use super::{TextOrImageElement, VisualElement};

#[derive(Debug, Clone, Copy)]
pub struct AttributionPlacement;

#[derive(Debug, Clone, Copy, Default)]
pub enum HintStyle {
  #[default]
  Default,
  Caption,
  CaptionSubtle,
  Body,
  BodySubtle,
  Base,
  BaseSubtle,
  Subtitle,
  SubtitleSubtle,
  Title,
  TitleSubtle,
  TitleNumeral,
  Subheader,
  SubheaderSubtle,
  SubheaderNumeral,
  Header,
  HeaderSubtle,
  HeaderNumeral,
}

impl ToString for HintStyle {
  fn to_string(&self) -> String {
    match self {
      HintStyle::Base => r#"hint-style="base""#.to_string(),
      HintStyle::Title => r#"hint-style="title""#.to_string(),
      HintStyle::Subtitle => r#"hint-style="subtitle""#.to_string(),
      HintStyle::CaptionSubtle => r#"hint-style="captionSubtle""#.to_string(),
      HintStyle::BaseSubtle => r#"hint-style="baseSubtle""#.to_string(),
      HintStyle::TitleSubtle => r#"hint-style="titleSubtle""#.to_string(),
      HintStyle::SubtitleSubtle => r#"hint-style="subtitleSubtle""#.to_string(),
      HintStyle::Caption => r#"hint-style="caption""#.to_string(),
      HintStyle::Body => r#"hint-style="body""#.to_string(),
      HintStyle::BodySubtle => r#"hint-style="bodySubtle""#.to_string(),
      HintStyle::Subheader => r#"hint-style="subheader""#.to_string(),
      HintStyle::SubheaderSubtle => r#"hint-style="subheaderSubtle""#.to_string(),
      HintStyle::SubheaderNumeral => r#"hint-style="subheaderNumeral""#.to_string(),
      HintStyle::Header => r#"hint-style="header""#.to_string(),
      HintStyle::HeaderSubtle => r#"hint-style="headerSubtle""#.to_string(),
      HintStyle::HeaderNumeral => r#"hint-style="headerNumeral""#.to_string(),
      HintStyle::Default => "".to_string(),
      HintStyle::TitleNumeral => "hint-style=\"titleNumeral\"".to_string(),
    }
  }
}

#[derive(Debug, Clone, Copy, Default)]
pub enum HintAlign {
  Right,
  Auto,
  Left,
  Center,
  #[default]
  Default,
}

impl ToString for HintAlign {
  fn to_string(&self) -> String {
    match self {
      HintAlign::Right => r#"hint-align="right""#.to_string(),
      HintAlign::Auto => r#"hint-align="auto""#.to_string(),
      HintAlign::Left => r#"hint-align="left""#.to_string(),
      HintAlign::Center => r#"hint-align="center""#.to_string(),
      HintAlign::Default => "".to_string(),
    }
  }
}

#[allow(non_snake_case)]
#[derive(Default)]
/// Learn more here
/// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-text>
pub struct Text {
  body: String,

  pub id: u64,
  pub lang: Option<String>,
  pub placement: Option<AttributionPlacement>,

  pub style: HintStyle,
  pub align: HintAlign,
  pub wrap: bool,
  pub callScenarioCenterAlign: bool,
  pub maxLines: u32,
  pub minLines: u32,
}

impl TextOrImageElement for Text {}

impl Text {
  pub fn create(id: u64, body: &str) -> Self {
    unsafe { Self::new_unchecked(id, None, None, escape(body).to_string()) }
  }

  pub fn create_binded(id: u64, binds: &str) -> Self {
    debug_assert!(binds.chars().all(|x| x.is_alphabetic()));

    unsafe { Self::new_unchecked(id, None, None, format!("{{{binds}}}")) }
  }

  pub fn with_align(mut self, align: HintAlign) -> Self {
    self.align = align;
    self
  }

  pub fn with_style(mut self, style: HintStyle) -> Self {
    self.style = style;
    self
  }

  pub fn with_lang(mut self, lang: String) -> Self {
    self.lang = Some(lang);
    self
  }

  pub fn with_placement(mut self, placement: AttributionPlacement) -> Self {
    self.placement = Some(placement);
    self
  }

  /// Only for IncomingCall scenarios
  pub fn with_align_center(mut self, shall_it_align_center: bool) -> Self {
    self.callScenarioCenterAlign = shall_it_align_center;
    self
  }

  pub fn with_wrap(mut self, wrap: bool) -> Self {
    self.wrap = wrap;
    self
  }

  #[deprecated(since="0.10.2", note="Use [Self::with_wrap] instead")]
  pub fn wrap(self, wrap: bool) -> Self {
    self.with_wrap(wrap)
  }

  pub fn with_max_lines(mut self, max_lines: u32) -> Self {
    self.maxLines = max_lines;
    self
  }

  pub fn with_min_lines(mut self, min_lines: u32) -> Self {
    self.minLines = min_lines;
    self
  }

  pub unsafe fn new_unchecked(
    id: u64,
    lang: Option<String>,
    placement: Option<AttributionPlacement>,
    body: String,
  ) -> Self {
    Self {
      id,
      lang,
      placement,
      body,
      ..Default::default()
    }
  }
}

impl VisualElement for Text {}

impl ToastVisualableXML for Text {}

impl ToXML for Text {
  fn to_xml(&self) -> String {
    format!(
      r#"
        <text id="{}" {} {} {} {} {} {} {} {}>
          {body}
        </text>
      "#,
      self.id,
      if self.wrap { "hint-wrap='true'" } else { "" },
      if self.maxLines > 0 {
        format!("hint-maxLines='{}'", self.maxLines)
      } else {
        "".to_string()
      },
      if self.minLines > 0 {
        format!("hint-minLines='{}'", self.minLines)
      } else {
        "".to_string()
      },
      if self.callScenarioCenterAlign {
        "hint-callScenarioCenterAlign='true'"
      } else {
        ""
      },
      self.align.to_string(),
      self.style.to_string(),
      self
        .lang
        .clone()
        .map_or_else(|| string!(""), |x| format!("lang=\"{x}\"")),
      self
        .placement
        .map_or_else(|| "", |_| "placement=\"attribution\""),
      body = self.body
    )
  }
}
