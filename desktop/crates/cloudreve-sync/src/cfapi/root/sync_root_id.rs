use std::{
    ffi::{OsStr, OsString},
    mem::MaybeUninit,
    os::windows::ffi::OsStringExt,
    path::Path,
    ptr,
};

use serde::{Deserialize, Serialize};
use widestring::{U16CStr, U16CString, U16Str, U16String, u16cstr};
use windows::{
    Storage::{Provider::StorageProviderSyncRootManager, StorageFolder},
    Win32::{
        Foundation::{
            self, ERROR_INSUFFICIENT_BUFFER, ERROR_INVALID_PARAMETER, HANDLE, HLOCAL, LocalFree,
        },
        Security::{self, Authorization::ConvertSidToStringSidW, GetTokenInformation, TOKEN_USER},
        Storage::CloudFilters,
        System::{
            Com::{self, CoCreateInstance},
            Search::{self, ISearchManager},
        },
    },
    core::{self, Error, HSTRING, PCWSTR, PWSTR},
};

use crate::cfapi::utility::ToHString;

use super::SyncRootInfo;

/// Returns a list of active sync roots.
pub fn active_roots() -> core::Result<Vec<SyncRootInfo>> {
    StorageProviderSyncRootManager::GetCurrentSyncRoots()
        .map(|list| list.into_iter().map(SyncRootInfo).collect())
}

/// Returns whether or not the Cloud Filter API is supported (or at least the UWP part of it, for
/// now).
pub fn is_supported() -> core::Result<bool> {
    StorageProviderSyncRootManager::IsSupported()
}

/// A builder to construct a [SyncRootId].
#[derive(Debug, Clone)]
pub struct SyncRootIdBuilder {
    provider_name: U16String,
    user_security_id: SecurityId,
    account_name: U16String,
}

impl SyncRootIdBuilder {
    /// Create a new builder with the given provider name.
    ///
    /// The provider name MUST NOT contain exclamation points and it must be within
    /// [255](https://docs.microsoft.com/en-us/windows/win32/api/cfapi/ns-cfapi-cf_sync_root_provider_info#remarks) characters.
    ///
    /// # Panics
    ///
    /// Panics if the provider name is longer than 255 characters or contains exclamation points.
    pub fn new(provider_name: impl AsRef<OsStr>) -> Self {
        let name = U16String::from_os_str(&provider_name);

        assert!(
            name.len() <= CloudFilters::CF_MAX_PROVIDER_NAME_LENGTH as usize,
            "provider name must not exceed {} characters, got {} characters",
            CloudFilters::CF_MAX_PROVIDER_NAME_LENGTH,
            name.len()
        );
        assert!(
            !name.as_slice().contains(&SyncRootId::SEPARATOR),
            "provider name must not contain exclamation points"
        );

        Self {
            provider_name: name,
            user_security_id: SecurityId(U16String::new()),
            account_name: U16String::new(),
        }
    }

    /// The security id of the Windows user. Retrieve this value via the
    /// [SecurityId] struct.
    ///
    /// By default, a sync root registered without a user security id will be installed globally.
    pub fn user_security_id(mut self, security_id: SecurityId) -> Self {
        self.user_security_id = security_id;
        self
    }

    /// The name of the user's account.
    ///
    /// This value does not have any actual meaning and it could theoretically be anything.
    /// However, it is encouraged to set this value to the account name of the user on the remote.
    pub fn account_name(mut self, account_name: impl AsRef<OsStr>) -> Self {
        self.account_name = U16String::from_os_str(&account_name);
        self
    }

    /// Constructs a [SyncRootId] from the builder.
    pub fn build(self) -> SyncRootId {
        SyncRootId(
            [
                self.provider_name.as_slice(),
                self.user_security_id.0.as_slice(),
                self.account_name.as_slice(),
            ]
            .join(&SyncRootId::SEPARATOR)
            .to_hstring(),
        )
    }
}

/// The identifier for a sync root.
///
/// The inner value comes in the form:
/// `provider-id!security-id!account-name`
/// as specified
/// [here](https://docs.microsoft.com/en-us/uwp/api/windows.storage.provider.storageprovidersyncrootinfo.id?view=winrt-22000#property-value).
///
/// A [SyncRootId] stores an inner, reference counted [HSTRING][windows::core::HSTRING], making this struct cheap to clone.
#[derive(Debug, Clone)]
pub struct SyncRootId(pub(crate) HSTRING);

impl Serialize for SyncRootId {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        // Convert HSTRING (UTF-16) to String (UTF-8) for serialization
        let os_string = OsString::from_wide(self.0.as_wide());
        let string = os_string.to_string_lossy();
        serializer.serialize_str(&string)
    }
}

impl<'de> Deserialize<'de> for SyncRootId {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        // Deserialize as String and convert to HSTRING
        let s = String::deserialize(deserializer)?;
        let u16_string = U16String::from_os_str(OsStr::new(&s));
        Ok(SyncRootId(u16_string.to_hstring()))
    }
}

impl SyncRootId {
    // https://docs.microsoft.com/en-us/uwp/api/windows.storage.provider.storageprovidersyncrootinfo.id?view=winrt-22000#windows-storage-provider-storageprovidersyncrootinfo-id
    // unicode exclamation point as told in the specification above
    const SEPARATOR: u16 = 0x21;

    /// Creates a [SyncRootId] from the sync root at the given path.
    pub fn from_path<P: AsRef<Path>>(path: P) -> core::Result<Self> {
        // if the id is coming from a sync root, then it must be valid
        StorageProviderSyncRootManager::GetSyncRootInformationForFolder(
            &StorageFolder::GetFolderFromPathAsync(
                &U16String::from_os_str(path.as_ref()).to_hstring(),
            )
            .unwrap()
            .get()?,
        )
        .map(|info| SyncRootId(info.Id().unwrap()))
    }

    /// Whether or not the [SyncRootId] has already been registered.
    pub fn is_registered(&self) -> core::Result<bool> {
        match StorageProviderSyncRootManager::GetSyncRootInformationForId(&self.0) {
            Ok(_) => Ok(true),
            Err(e) if e.code() == Foundation::ERROR_NOT_FOUND.to_hresult() => Ok(false),
            Err(e) => Err(e),
        }
    }

    /// Returns the sync root information for the [SyncRootId].
    pub fn info(&self) -> core::Result<SyncRootInfo> {
        StorageProviderSyncRootManager::GetSyncRootInformationForId(&self.0).map(SyncRootInfo)
    }

    /// Registers the sync root at the current [SyncRootId].
    ///
    /// [SyncRootInfo::display_name], [SyncRootInfo::icon], [SyncRootInfo::version] and [SyncRootInfo::path]
    /// are required and cannot be empty.
    pub fn register(&self, info: SyncRootInfo) -> core::Result<()> {
        macro_rules! check_field {
            ($info:ident, $field:ident) => {
                if $info.$field().eq(OsStr::new("")) {
                    Err(Error::new(
                        ERROR_INVALID_PARAMETER.to_hresult(),
                        concat!(stringify!($field), " cannot be empty"),
                    ))?;
                }
            };
        }
        check_field!(info, display_name);
        check_field!(info, icon);
        check_field!(info, version);
        check_field!(info, path);

        info.0.SetId(&self.0).unwrap();
        println!("StorageProviderSyncRootManager Register ID: {:?}", &self.0);
        StorageProviderSyncRootManager::Register(&info.0)
    }

    /// Unregisters the sync root at the current [SyncRootId] if it exists.
    pub fn unregister(&self) -> core::Result<()> {
        StorageProviderSyncRootManager::Unregister(&self.0)
    }

    /// Indexes the sync root at the current [SyncRootId].
    ///
    /// Returns an error if the sync root does not exist or unable to index.
    pub fn index(&self) -> core::Result<()> {
        let path = self.info()?.path();
        index_path(&path)
    }

    /// Encodes the [SyncRootId] to an [OsString].
    pub fn to_os_string(&self) -> OsString {
        OsString::from_wide(self.0.as_wide())
    }

    /// A reference to the [SyncRootId] as a 16 bit string.
    pub fn as_u16_str(&self) -> &U16Str {
        U16Str::from_slice(self.0.as_wide())
    }

    /// A reference to the [SyncRootId] as an [HSTRING][windows::core::HSTRING] (its inner value).
    pub fn as_hstring(&self) -> &HSTRING {
        &self.0
    }

    /// The three components of a [SyncRootId] as described by the specification.
    ///
    /// The order goes as follows:
    /// `(provider-id, security-id, account-name)`
    ///
    /// # Panics
    ///
    /// Panics if the sync root id does not have exactly three components.
    pub fn to_components(&self) -> (&U16Str, &U16Str, &U16Str) {
        let mut components = Vec::with_capacity(3);
        components.extend(
            self.0
                .as_wide()
                .split(|&byte| byte == Self::SEPARATOR)
                .map(U16Str::from_slice),
        );

        if components.len() != 3 {
            panic!("malformed sync root id, got {:?}", components)
        }

        (components[0], components[1], components[2])
    }
}

/// A user security id (SID).
#[derive(Debug, Clone)]
pub struct SecurityId(U16String);

impl SecurityId {
    // https://docs.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-getcurrentthreadeffectivetoken
    const CURRENT_THREAD_EFFECTIVE_TOKEN: HANDLE = HANDLE(-6isize as *mut ::core::ffi::c_void);

    /// Creates a new [SecurityId] from [OsString].
    ///
    /// # Panics
    ///
    /// Panics if the security id contains an exclamation point.
    pub fn new(id: impl AsRef<OsStr>) -> Self {
        let id = U16String::from_os_str(&id);
        assert!(
            !id.as_slice().contains(&SyncRootId::SEPARATOR),
            "security id cannot contain exclamation points"
        );

        Self(id)
    }

    /// The [SecurityId] for the logged in user.
    pub fn current_user() -> core::Result<Self> {
        unsafe {
            let mut token_size = 0;

            // get the token size
            let info = GetTokenInformation(
                Self::CURRENT_THREAD_EFFECTIVE_TOKEN,
                Security::TokenUser,
                None,
                0,
                &mut token_size,
            );

            if let Err(e) = info {
                if e.code() != ERROR_INSUFFICIENT_BUFFER.to_hresult() {
                    Err(e)?;
                }
            }

            let mut buffer = Vec::<MaybeUninit<u8>>::with_capacity(token_size as usize);
            buffer.set_len(token_size as usize);

            GetTokenInformation(
                Self::CURRENT_THREAD_EFFECTIVE_TOKEN,
                Security::TokenUser,
                Some(buffer.as_mut_ptr() as *mut _),
                token_size,
                &mut token_size,
            )?;

            let token_user = &*(buffer.as_ptr() as *const TOKEN_USER);
            let mut sid = PWSTR(ptr::null_mut());
            ConvertSidToStringSidW(token_user.User.Sid, &mut sid as *mut _)?;

            let string_sid = U16CStr::from_ptr_str(sid.0).to_os_string();
            LocalFree(HLOCAL(sid.0 as *mut _));

            Ok(SecurityId::new(string_sid))
        }
    }
}

fn index_path(path: &Path) -> core::Result<()> {
    unsafe {
        let searcher: ISearchManager = CoCreateInstance(
            &Search::CSearchManager as *const _,
            None,
            Com::CLSCTX_SERVER,
        )?;

        let catalog = searcher.GetCatalog(PCWSTR(u16cstr!("SystemIndex").as_ptr()))?;

        let mut url = OsString::from("file:///");
        url.push(path);

        let crawler = catalog.GetCrawlScopeManager()?;
        crawler.AddDefaultScopeRule(
            PCWSTR(
                U16CString::from_os_str(url)
                    .expect("not contains nul")
                    .as_ptr(),
            ),
            true,
            Search::FF_INDEXCOMPLEXURLS.0 as u32,
        )?;

        crawler.SaveAll()
    }
}
