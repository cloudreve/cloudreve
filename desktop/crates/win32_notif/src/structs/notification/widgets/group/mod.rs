mod group;
mod subgroup;

pub use group::*;
pub use subgroup::*;

use crate::ToXML;

pub trait SubgroupXML: ToXML {}
