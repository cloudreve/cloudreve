use std::{
    ffi::{OsStr, OsString},
    os::windows::ffi::OsStringExt,
    path::{Path, PathBuf},
};

use crate::cfapi::utility::ToHString;
use anyhow::Result;
use flagset::{FlagSet, flags};
use widestring::U16String;
use windows::{
    Foundation::Uri,
    Storage::{
        Provider::{
            StorageProviderHardlinkPolicy, StorageProviderHydrationPolicy,
            StorageProviderHydrationPolicyModifier, StorageProviderInSyncPolicy,
            StorageProviderItemPropertyDefinition, StorageProviderPopulationPolicy,
            StorageProviderProtectionMode, StorageProviderSyncRootInfo,
        },
        StorageFolder,
        Streams::{DataReader, DataWriter},
    },
    core,
};

use super::SyncRootId;

#[derive(Clone)]
pub struct SyncRootInfo(pub(crate) StorageProviderSyncRootInfo);

impl SyncRootInfo {
    /// Enables or disables the ability for files to be made available offline.
    pub fn allow_pinning(&self) -> bool {
        self.0.AllowPinning().unwrap()
    }

    /// Sets the ability for files to be made available offline.
    pub fn set_allow_pinning(&mut self, allow_pinning: bool) {
        self.0.SetAllowPinning(allow_pinning).unwrap()
    }

    /// Sets the ability for files to be made available offline.
    pub fn with_allow_pinning(mut self, allow_pinning: bool) -> Self {
        self.set_allow_pinning(allow_pinning);
        self
    }

    /// Hard links are allowed on a placeholder within the same sync root.
    pub fn allow_hardlinks(&self) -> bool {
        self.0.HardlinkPolicy().unwrap() == StorageProviderHardlinkPolicy::Allowed
    }

    /// Sets the hard link are allowed on a placeholder within the same sync root.
    pub fn set_allow_hardlinks(&mut self, allow_hardlinks: bool) {
        self.0
            .SetHardlinkPolicy(if allow_hardlinks {
                StorageProviderHardlinkPolicy::Allowed
            } else {
                StorageProviderHardlinkPolicy::None
            })
            .unwrap()
    }

    /// Sets the hard link are allowed on a placeholder within the same sync root.
    pub fn with_allow_hardlinks(mut self, allow_hardlinks: bool) -> Self {
        self.set_allow_hardlinks(allow_hardlinks);
        self
    }

    /// An optional display name that maps to the existing sync root registration.
    pub fn display_name(&self) -> OsString {
        self.0.DisplayNameResource().unwrap().to_os_string()
    }

    /// Sets the display name that maps to the existing sync root registration.
    pub fn set_display_name(&mut self, display_name: impl AsRef<OsStr>) {
        self.0
            .SetDisplayNameResource(&U16String::from_os_str(&display_name).to_hstring())
            .unwrap()
    }

    pub fn add_custom_state(&mut self, display_name: impl AsRef<OsStr>, id: i32) -> Result<()> {
        let property_definition = StorageProviderItemPropertyDefinition::new()?;
        property_definition.SetId(id)?;
        property_definition
            .SetDisplayNameResource(&U16String::from_os_str(&display_name).to_hstring())?;
        self.0
            .StorageProviderItemPropertyDefinitions()?
            .Append(&property_definition)?;
        Ok(())
    }

    /// Sets the display name that maps to the existing sync root registration.
    pub fn with_display_name(mut self, display_name: impl AsRef<OsStr>) -> Self {
        self.set_display_name(display_name);
        self
    }

    /// A Uri to a cloud storage recycle bin.
    pub fn recycle_bin_uri(&self) -> Option<OsString> {
        self.0
            .RecycleBinUri()
            .map(|uri| uri.ToString().unwrap().to_os_string())
            .ok()
    }

    /// Sets the Uri to a cloud storage recycle bin.
    ///
    /// Returns an error if the Uri is not valid.
    pub fn set_recycle_bin_uri(&mut self, recycle_bin_uri: impl AsRef<OsStr>) -> core::Result<()> {
        self.0
            .SetRecycleBinUri(&Uri::CreateUri(
                &U16String::from_os_str(&recycle_bin_uri).to_hstring(),
            )?)
            .unwrap();

        Ok(())
    }

    /// Sets the Uri to a cloud storage recycle bin.
    ///
    /// Returns an error if the Uri is not valid.
    pub fn with_recycle_bin_uri(
        mut self,
        recycle_bin_uri: impl AsRef<OsStr>,
    ) -> core::Result<Self> {
        self.set_recycle_bin_uri(recycle_bin_uri)?;
        Ok(self)
    }

    /// Shows sibling sync roots listed under the main sync root in the File Explorer.
    pub fn show_siblings_as_group(&self) -> bool {
        self.0.ShowSiblingsAsGroup().unwrap()
    }

    /// Shows sibling sync roots listed under the main sync root in the File Explorer or not.
    pub fn set_show_siblings_as_group(&mut self, show_siblings_as_group: bool) {
        self.0
            .SetShowSiblingsAsGroup(show_siblings_as_group)
            .unwrap()
    }

    /// Shows sibling sync roots listed under the main sync root in the File Explorer or not.
    pub fn with_show_siblings_as_group(mut self, show_siblings_as_group: bool) -> Self {
        self.set_show_siblings_as_group(show_siblings_as_group);
        self
    }

    /// The path of the sync root.
    pub fn path(&self) -> PathBuf {
        self.0
            .Path()
            .map(|path| path.Path().unwrap().to_os_string().into())
            .unwrap_or_default()
    }

    /// Sets the path of the sync root.
    ///
    /// Returns an error if the path is not a folder.
    pub fn set_path(&mut self, path: impl AsRef<Path>) -> core::Result<()> {
        self.0
            .SetPath(
                &StorageFolder::GetFolderFromPathAsync(
                    &U16String::from_os_str(path.as_ref()).to_hstring(),
                )
                .unwrap()
                .get()?,
            )
            .unwrap();
        Ok(())
    }

    /// Sets the path of the sync root.
    ///
    /// Returns an error if the path is not a folder.
    pub fn with_path(mut self, path: impl AsRef<Path>) -> core::Result<Self> {
        self.set_path(path)?;
        Ok(self)
    }

    /// The population policy of the sync root registration.
    pub fn population_type(&self) -> PopulationType {
        self.0.PopulationPolicy().unwrap().into()
    }

    /// Sets the population policy of the sync root registration.
    pub fn set_population_type(&mut self, population_type: PopulationType) {
        self.0.SetPopulationPolicy(population_type.into()).unwrap();
    }

    /// Sets the population policy of the sync root registration.
    pub fn with_population_type(mut self, population_type: PopulationType) -> Self {
        self.set_population_type(population_type);
        self
    }

    /// The version number of the sync root provider.
    pub fn version(&self) -> OsString {
        OsString::from_wide(self.0.Version().unwrap().as_wide())
    }

    /// Sets the version number of the sync root provider.
    pub fn set_version(&mut self, version: impl AsRef<OsStr>) {
        self.0
            .SetVersion(&U16String::from_os_str(&version).to_hstring())
            .unwrap()
    }

    /// Sets the version number of the sync root provider.
    pub fn with_version(mut self, version: impl AsRef<OsStr>) -> Self {
        self.set_version(version);
        self
    }

    /// The protection mode of the sync root registration.
    pub fn protection_mode(&self) -> ProtectionMode {
        self.0.ProtectionMode().unwrap().into()
    }

    /// Sets the protection mode of the sync root registration.
    pub fn set_protection_mode(&mut self, protection_mode: ProtectionMode) {
        self.0.SetProtectionMode(protection_mode.into()).unwrap();
    }

    /// Sets the protection mode of the sync root registration.
    pub fn with_protection_mode(mut self, protection_mode: ProtectionMode) -> Self {
        self.set_protection_mode(protection_mode);
        self
    }

    /// The supported attributes of the sync root registration.
    pub fn supported_attribute(&self) -> FlagSet<SupportedAttribute> {
        FlagSet::new(self.0.InSyncPolicy().unwrap().0).expect("flags should be valid")
    }

    /// Sets the supported attributes of the sync root registration.
    pub fn set_supported_attribute(
        &mut self,
        supported_attribute: impl Into<FlagSet<SupportedAttribute>>,
    ) {
        self.0
            .SetInSyncPolicy(StorageProviderInSyncPolicy(
                supported_attribute.into().bits(),
            ))
            .unwrap();
    }

    /// Sets the supported attributes of the sync root registration.
    pub fn with_supported_attribute(
        mut self,
        supported_attribute: impl Into<FlagSet<SupportedAttribute>>,
    ) -> Self {
        self.set_supported_attribute(supported_attribute);
        self
    }

    /// The hydration policy of the sync root registration.
    pub fn hydration_type(&self) -> HydrationType {
        self.0.HydrationPolicy().unwrap().into()
    }

    /// Sets the hydration policy of the sync root registration.
    pub fn set_hydration_type(&mut self, hydration_type: HydrationType) {
        self.0.SetHydrationPolicy(hydration_type.into()).unwrap();
    }

    /// Sets the hydration policy of the sync root registration.
    pub fn with_hydration_type(mut self, hydration_type: HydrationType) -> Self {
        self.set_hydration_type(hydration_type);
        self
    }

    /// The hydration policy of the sync root registration.
    pub fn hydration_policy(&self) -> FlagSet<HydrationPolicy> {
        FlagSet::new(self.0.HydrationPolicyModifier().unwrap().0).expect("flags should be valid")
    }

    /// Sets the hydration policy of the sync root registration.
    pub fn set_hydration_policy(&mut self, hydration_policy: impl Into<FlagSet<HydrationPolicy>>) {
        self.0
            .SetHydrationPolicyModifier(StorageProviderHydrationPolicyModifier(
                hydration_policy.into().bits(),
            ))
            .unwrap();
    }

    /// Sets the hydration policy of the sync root registration.
    pub fn with_hydration_policy(
        mut self,
        hydration_policy: impl Into<FlagSet<HydrationPolicy>>,
    ) -> Self {
        self.set_hydration_policy(hydration_policy);
        self
    }

    /// The icon of the sync root registration.
    pub fn icon(&self) -> OsString {
        self.0.IconResource().unwrap().to_os_string()
    }

    /// Sets the icon of the sync root registration.
    ///
    /// See also <https://docs.microsoft.com/en-us/windows/win32/menurc/icon-resource>.
    pub fn set_icon(&mut self, icon: impl AsRef<OsStr>) {
        self.0
            .SetIconResource(&U16String::from_os_str(&icon).to_hstring())
            .unwrap();
    }

    /// Sets the icon of the sync root registration.
    ///
    /// See also <https://docs.microsoft.com/en-us/windows/win32/menurc/icon-resource>.
    pub fn with_icon(mut self, icon: impl AsRef<OsStr>) -> Self {
        self.set_icon(icon);
        self
    }

    /// The identifier of the sync root registration.
    pub fn id(&self) -> SyncRootId {
        SyncRootId(self.0.Id().unwrap())
    }

    /// The blob of the sync root registration.
    pub fn blob(&self) -> Vec<u8> {
        let Ok(buffer) = self.0.Context() else {
            return Vec::new();
        };
        let mut data = vec![0u8; buffer.Length().unwrap() as usize];
        let reader = DataReader::FromBuffer(&buffer).unwrap();
        reader.ReadBytes(data.as_mut_slice()).unwrap();

        data
    }

    /// Sets the blob of the sync root registration.
    pub fn set_blob(&mut self, blob: &[u8]) {
        let writer = DataWriter::new().unwrap();
        writer.WriteBytes(blob).unwrap();
        self.0.SetContext(&writer.DetachBuffer().unwrap()).unwrap();
    }

    /// Sets the blob of the sync root registration.
    pub fn with_blob(mut self, blob: &[u8]) -> Self {
        self.set_blob(blob);
        self
    }
}

impl Default for SyncRootInfo {
    fn default() -> Self {
        Self(StorageProviderSyncRootInfo::new().unwrap())
    }
}

/// The protection mode of the sync root registration.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ProtectionMode {
    /// The sync root should only contain personal files, not encrypted or business related files.
    Personal,
    /// The sync root can contain any type of file.
    Unknown,
}

impl From<ProtectionMode> for StorageProviderProtectionMode {
    fn from(mode: ProtectionMode) -> Self {
        match mode {
            ProtectionMode::Personal => StorageProviderProtectionMode::Personal,
            ProtectionMode::Unknown => StorageProviderProtectionMode::Unknown,
        }
    }
}

impl From<StorageProviderProtectionMode> for ProtectionMode {
    fn from(mode: StorageProviderProtectionMode) -> Self {
        match mode {
            StorageProviderProtectionMode::Personal => ProtectionMode::Personal,
            StorageProviderProtectionMode::Unknown => ProtectionMode::Unknown,
            _ => unreachable!(),
        }
    }
}

flags! {
    /// Attributes supported by the sync root.
    pub enum SupportedAttribute: u32 {
        FileCreationTime = StorageProviderInSyncPolicy::FileCreationTime.0,
        FileReadonly = StorageProviderInSyncPolicy::FileReadOnlyAttribute.0,
        FileHidden = StorageProviderInSyncPolicy::FileHiddenAttribute.0,
        FileSystem = StorageProviderInSyncPolicy::FileSystemAttribute.0,
        FileLastWriteTime = StorageProviderInSyncPolicy::FileLastWriteTime.0,
        DirectoryCreationTime = StorageProviderInSyncPolicy::DirectoryCreationTime.0,
        DirectoryReadonly = StorageProviderInSyncPolicy::DirectoryReadOnlyAttribute.0,
        DirectoryHidden = StorageProviderInSyncPolicy::DirectoryHiddenAttribute.0,
        DirectoryLastWriteTime = StorageProviderInSyncPolicy::DirectoryLastWriteTime.0,
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum HydrationType {
    Partial,
    Progressive,
    Full,
    AlwaysFull,
}

impl From<HydrationType> for StorageProviderHydrationPolicy {
    fn from(hydration_type: HydrationType) -> Self {
        match hydration_type {
            HydrationType::Partial => StorageProviderHydrationPolicy::Partial,
            HydrationType::Progressive => StorageProviderHydrationPolicy::Progressive,
            HydrationType::Full => StorageProviderHydrationPolicy::Full,
            HydrationType::AlwaysFull => StorageProviderHydrationPolicy::AlwaysFull,
        }
    }
}

impl From<StorageProviderHydrationPolicy> for HydrationType {
    fn from(policy: StorageProviderHydrationPolicy) -> Self {
        match policy {
            StorageProviderHydrationPolicy::Partial => HydrationType::Partial,
            StorageProviderHydrationPolicy::Progressive => HydrationType::Progressive,
            StorageProviderHydrationPolicy::Full => HydrationType::Full,
            StorageProviderHydrationPolicy::AlwaysFull => HydrationType::AlwaysFull,
            _ => unreachable!(),
        }
    }
}

flags! {
    /// Hydration policy
    pub enum HydrationPolicy: u32 {
        ValidationRequired = StorageProviderHydrationPolicyModifier::ValidationRequired.0,
        StreamingAllowed = StorageProviderHydrationPolicyModifier::StreamingAllowed.0,
        AutoDehydrationAllowed = StorageProviderHydrationPolicyModifier::AutoDehydrationAllowed.0,
        AllowFullRestartHydration = StorageProviderHydrationPolicyModifier::AllowFullRestartHydration.0,
    }
}

/// The population policy of the sync root registration.
#[derive(Debug, Clone, Copy)]
pub enum PopulationType {
    /// If the placeholder files or directories are not fully populated,
    /// the platform will request that the sync provider populate them before completing a user request.
    Full,
    /// The platform will assume that placeholder files and directories are always available locally.
    AlwaysFull,
}

impl From<PopulationType> for StorageProviderPopulationPolicy {
    fn from(population_type: PopulationType) -> StorageProviderPopulationPolicy {
        match population_type {
            PopulationType::Full => StorageProviderPopulationPolicy::Full,
            PopulationType::AlwaysFull => StorageProviderPopulationPolicy::AlwaysFull,
        }
    }
}

impl From<StorageProviderPopulationPolicy> for PopulationType {
    fn from(population_type: StorageProviderPopulationPolicy) -> Self {
        match population_type {
            StorageProviderPopulationPolicy::Full => PopulationType::Full,
            StorageProviderPopulationPolicy::AlwaysFull => PopulationType::AlwaysFull,
            _ => unreachable!(),
        }
    }
}

impl std::fmt::Debug for SyncRootInfo {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("SyncRootInfo")
            .field("allow_pinning", &self.allow_pinning())
            .field("allow_hardlinks", &self.allow_hardlinks())
            .field("display name", &self.display_name())
            .field("recycle_bin_uri", &self.recycle_bin_uri())
            .field("hydration_policy", &self.hydration_policy())
            .field("hydration_type", &self.hydration_type())
            .field("icon", &self.icon())
            .field("path", &self.path())
            .field("population_type", &self.population_type())
            .field("protection_mode", &self.protection_mode())
            .field("supported_attribute", &self.supported_attribute())
            .field("show_siblings_as_group", &self.show_siblings_as_group())
            .field("id", &self.id())
            .field("version", &self.version())
            .finish()
    }
}
