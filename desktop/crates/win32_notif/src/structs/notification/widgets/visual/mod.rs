pub mod image;
pub mod progress;
pub mod text;

use crate::ToXML;

pub trait VisualElement {}
pub trait TextOrImageElement: VisualElement + ToXML {}

pub use image::{Image, Placement};
pub use progress::Progress;
pub use text::{AttributionPlacement, Text};
