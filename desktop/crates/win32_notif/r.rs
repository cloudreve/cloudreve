#![feature(prelude_import)]
#![allow(private_bounds)]
//! Win32 Notification
//!
//! This library implements UWP XML Toast Notification
//! This is a safe wrapper around the official WinRT apis
//!
//! # Example
//! ```rust
//! use win32_notif::{
//!  notification::visual::progress::{Progress, ProgressValue},
//!  string, NotificationBuilder, ToastsNotifier,
//! };
//!
//! fn main() {
//!   let notifier = ToastsNotifier::new("Microsoft.Windows.Explorer").unwrap();
//!   let notif = NotificationBuilder::new()
//!     .visual(Progress::new(
//!       None,
//!       string!("Downloading..."),
//!       ProgressValue::BindTo("prog"),
//!       None,
//!     ))
//!     // Use the newest data binding method
//!     .value("prog", "0.3")
//!     .build(1, &notifier, "a", "ahq")
//!     .unwrap();
//!
//!   let _ = notif.show();
//!   loop {}
//! }
//! ```
#[macro_use]
extern crate std;
#[prelude_import]
use std::prelude::rust_2021::*;
mod structs {
    pub mod data {
        use windows::UI::Notifications::NotificationData;
        use crate::NotifError;
        pub struct NotificationDataSet {
            _inner: NotificationData,
        }
        impl NotificationDataSet {
            pub fn new() -> Result<Self, NotifError> {
                Ok(Self {
                    _inner: NotificationData::new()?,
                })
            }
            pub fn insert(&self, k: &str, v: &str) -> Result<bool, NotifError> {
                Ok(self._inner.Values()?.Insert(&k.into(), &v.into())?)
            }
            pub fn inner_win32_type(&self) -> &NotificationData {
                &self._inner
            }
        }
    }
    pub mod handler {
        pub mod activated {
            use std::collections::HashMap;
            use windows::{
                core::{Error, IInspectable, Interface, Ref, HSTRING},
                Foundation::{IReference, TypedEventHandler},
                UI::Notifications::{ToastActivatedEventArgs, ToastNotification},
            };
            use crate::notification::PartialNotification;
            pub struct ToastActivatedArgs {
                pub button_id: Option<String>,
                pub user_input: Option<HashMap<String, String>>,
            }
            #[automatically_derived]
            impl ::core::fmt::Debug for ToastActivatedArgs {
                #[inline]
                fn fmt(&self, f: &mut ::core::fmt::Formatter) -> ::core::fmt::Result {
                    ::core::fmt::Formatter::debug_struct_field2_finish(
                        f,
                        "ToastActivatedArgs",
                        "button_id",
                        &self.button_id,
                        "user_input",
                        &&self.user_input,
                    )
                }
            }
            impl ToastActivatedArgs {
                pub(crate) fn new(args: ToastActivatedEventArgs) -> Self {
                    let argument = args
                        .Arguments()
                        .ok()
                        .and_then(|x| Some(x.to_string()));
                    let user_input = args
                        .UserInput()
                        .ok()
                        .and_then(|x| Some(x.into_iter()));
                    let user_input = user_input
                        .and_then(|x| {
                            let mut val: HashMap<String, String> = HashMap::new();
                            x.for_each(|x| {
                                let _: Option<()> = (|| {
                                    let key = x.Key().ok()?;
                                    let key = key.to_string();
                                    let value = x.Value().ok()?;
                                    let value = value.cast::<IReference<HSTRING>>().ok();
                                    let value = value?.GetString().ok()?;
                                    let value = value.to_string();
                                    let _ = val.insert(key, value);
                                    Some(())
                                })();
                            });
                            Some(val)
                        });
                    Self {
                        button_id: argument,
                        user_input,
                    }
                }
            }
            pub struct NotificationActivatedEventHandler {
                pub(crate) handler: TypedEventHandler<ToastNotification, IInspectable>,
            }
            impl NotificationActivatedEventHandler {
                pub fn new<
                    T: Fn(
                            Option<PartialNotification>,
                            Option<ToastActivatedArgs>,
                        ) -> Result<(), Error> + Send + Sync + 'static,
                >(func: T) -> Self {
                    let handler: TypedEventHandler<ToastNotification, IInspectable> = TypedEventHandler::new(move |
                        a: Ref<ToastNotification>,
                        b: Ref<IInspectable>|
                    {
                        let a = a.as_ref();
                        let a = a.and_then(|a| PartialNotification { _toast: a }.into());
                        let b = b.as_ref();
                        let b = b.and_then(|x| x.cast::<ToastActivatedEventArgs>().ok());
                        let b = b.and_then(|x| Some(ToastActivatedArgs::new(x)));
                        func(a, b)
                    });
                    Self { handler }
                }
            }
        }
        pub mod dismissed {
            use windows::{
                core::{Error, Interface, Ref},
                Foundation::TypedEventHandler,
                UI::Notifications::{
                    ToastDismissalReason, ToastDismissedEventArgs, ToastNotification,
                },
            };
            use crate::notification::PartialNotification;
            pub enum ToastDismissedReason {
                Unknown(String),
                UserCanceled,
                ApplicationHidden,
                TimedOut,
            }
            #[automatically_derived]
            impl ::core::fmt::Debug for ToastDismissedReason {
                #[inline]
                fn fmt(&self, f: &mut ::core::fmt::Formatter) -> ::core::fmt::Result {
                    match self {
                        ToastDismissedReason::Unknown(__self_0) => {
                            ::core::fmt::Formatter::debug_tuple_field1_finish(
                                f,
                                "Unknown",
                                &__self_0,
                            )
                        }
                        ToastDismissedReason::UserCanceled => {
                            ::core::fmt::Formatter::write_str(f, "UserCanceled")
                        }
                        ToastDismissedReason::ApplicationHidden => {
                            ::core::fmt::Formatter::write_str(f, "ApplicationHidden")
                        }
                        ToastDismissedReason::TimedOut => {
                            ::core::fmt::Formatter::write_str(f, "TimedOut")
                        }
                    }
                }
            }
            impl ToastDismissedReason {
                pub(crate) fn new(args: ToastDismissedEventArgs) -> Self {
                    args.Reason()
                        .map_or_else(
                            |x| Self::Unknown(x.message()),
                            |x| {
                                let x = x.0;
                                if x == ToastDismissalReason::ApplicationHidden.0 {
                                    Self::ApplicationHidden
                                } else if x == ToastDismissalReason::UserCanceled.0 {
                                    Self::UserCanceled
                                } else if x == ToastDismissalReason::TimedOut.0 {
                                    Self::TimedOut
                                } else {
                                    Self::Unknown(
                                        ::alloc::__export::must_use({
                                            ::alloc::fmt::format(format_args!("Unknown reason: {0}", x))
                                        }),
                                    )
                                }
                            },
                        )
                }
            }
            pub struct NotificationDismissedEventHandler {
                pub(crate) handler: TypedEventHandler<
                    ToastNotification,
                    ToastDismissedEventArgs,
                >,
            }
            impl NotificationDismissedEventHandler {
                pub fn new<
                    T: Fn(
                            Option<PartialNotification>,
                            Option<ToastDismissedReason>,
                        ) -> Result<(), Error> + Send + Sync + 'static,
                >(func: T) -> Self {
                    let handler: TypedEventHandler<
                        ToastNotification,
                        ToastDismissedEventArgs,
                    > = TypedEventHandler::new(move |
                        a: Ref<ToastNotification>,
                        b: Ref<ToastDismissedEventArgs>|
                    {
                        let a = a.as_ref();
                        let a = a.and_then(|a| PartialNotification { _toast: a }.into());
                        let b = b.as_ref();
                        let b = b.and_then(|x| x.cast::<ToastDismissedEventArgs>().ok());
                        let b = b.and_then(|x| Some(ToastDismissedReason::new(x)));
                        func(a, b)
                    });
                    Self { handler }
                }
            }
        }
        pub mod failed {
            use windows::{
                core::{Error, Interface, Ref},
                Foundation::TypedEventHandler,
                UI::Notifications::{ToastFailedEventArgs, ToastNotification},
            };
            use crate::notification::PartialNotification;
            pub struct ToastFailedArgs {
                pub error: Option<String>,
            }
            #[automatically_derived]
            impl ::core::fmt::Debug for ToastFailedArgs {
                #[inline]
                fn fmt(&self, f: &mut ::core::fmt::Formatter) -> ::core::fmt::Result {
                    ::core::fmt::Formatter::debug_struct_field1_finish(
                        f,
                        "ToastFailedArgs",
                        "error",
                        &&self.error,
                    )
                }
            }
            impl ToastFailedArgs {
                pub(crate) fn new(args: ToastFailedEventArgs) -> Self {
                    Self {
                        error: args.ErrorCode().ok().and_then(|x| x.to_string().into()),
                    }
                }
            }
            pub struct NotificationFailedEventHandler {
                pub(crate) handler: TypedEventHandler<
                    ToastNotification,
                    ToastFailedEventArgs,
                >,
            }
            impl NotificationFailedEventHandler {
                pub fn new<
                    T: Fn(
                            Option<PartialNotification>,
                            Option<ToastFailedArgs>,
                        ) -> Result<(), Error> + Send + Sync + 'static,
                >(func: T) -> Self {
                    let handler: TypedEventHandler<
                        ToastNotification,
                        ToastFailedEventArgs,
                    > = TypedEventHandler::new(move |
                        a: Ref<ToastNotification>,
                        b: Ref<ToastFailedEventArgs>|
                    {
                        let a = a.as_ref();
                        let a = a.and_then(|a| PartialNotification { _toast: a }.into());
                        let b = b.as_ref();
                        let b = b.and_then(|x| x.cast::<ToastFailedEventArgs>().ok());
                        let b = b.and_then(|x| Some(ToastFailedArgs::new(x)));
                        func(a, b)
                    });
                    Self { handler }
                }
            }
        }
        pub use activated::{NotificationActivatedEventHandler, ToastActivatedArgs};
        pub use dismissed::{NotificationDismissedEventHandler, ToastDismissedReason};
        pub use failed::{NotificationFailedEventHandler, ToastFailedArgs};
    }
    pub mod notification {
        use std::collections::HashMap;
        use crate::NotifError;
        use super::{
            handler::{NotificationDismissedEventHandler, NotificationFailedEventHandler},
            NotificationActivatedEventHandler, NotificationImpl, ToXML, ToastsNotifier,
        };
        use actions::ActionElement;
        use audio::Audio;
        use header::Header;
        use visual::VisualElement;
        use widgets::commands::Commands;
        use windows::{
            core::HSTRING, Data::Xml::Dom::XmlDocument,
            Foundation::{DateTime, IReference, PropertyValue},
            Globalization::Calendar,
            UI::Notifications::{NotificationData, ToastNotification},
        };
        use windows_core::Interface;
        use std::time::Duration;
        mod widgets {
            use quick_xml::escape::escape;
            pub mod actions {
                pub mod action {
                    use quick_xml::escape::escape;
                    use crate::{notification::ActionableXML, ToXML};
                    use super::ActionElement;
                    #[allow(non_snake_case)]
                    /// Learn More Here
                    /// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-action>
                    pub struct ActionButton {
                        content: String,
                        arguments: String,
                        imageUri: Option<String>,
                        hint_inputid: String,
                        hint_toolTip: String,
                        activationType: String,
                        afterActivationBehavior: String,
                        hint_buttonStyle: String,
                        placement: bool,
                    }
                    #[allow(non_snake_case)]
                    impl ActionButton {
                        pub fn create<T: AsRef<str>>(content: T) -> Self {
                            unsafe {
                                Self::new_unchecked(
                                    escape(content.as_ref()).into(),
                                    escape(content.as_ref()).into(),
                                    ActivationType::Foreground,
                                    AfterActivationBehavior::Default,
                                    None,
                                    "".into(),
                                    HintButtonStyle::None,
                                    "".into(),
                                    false,
                                )
                            }
                        }
                        pub fn with_id(mut self, id: &str) -> Self {
                            self.arguments = escape(id).into();
                            self
                        }
                        /// Provide input id to place the button near an input
                        pub fn with_input_id(mut self, id: &str) -> Self {
                            self.hint_inputid = escape(id).into();
                            self
                        }
                        pub fn with_tooltip(mut self, tooltip: &str) -> Self {
                            self.hint_toolTip = escape(tooltip).into();
                            self
                        }
                        pub fn with_image_uri(mut self, uri: &str) -> Self {
                            self.imageUri = Some(escape(uri).into());
                            self
                        }
                        pub fn with_context_menu_placement(
                            mut self,
                            enabled: bool,
                        ) -> Self {
                            self.placement = enabled;
                            self
                        }
                        pub fn with_activation_type(
                            mut self,
                            activation_type: ActivationType,
                        ) -> Self {
                            self.activationType = activation_type.into();
                            self
                        }
                        pub fn with_after_activation_behavior(
                            mut self,
                            after_activation_behavior: AfterActivationBehavior,
                        ) -> Self {
                            self.afterActivationBehavior = after_activation_behavior
                                .into();
                            self
                        }
                        pub fn with_button_style(
                            mut self,
                            hint_buttonStyle: HintButtonStyle,
                        ) -> Self {
                            self.hint_buttonStyle = hint_buttonStyle.into();
                            self
                        }
                        pub fn with_content(mut self, content: &str) -> Self {
                            self.content = escape(content).into();
                            self
                        }
                        pub unsafe fn new_unchecked(
                            content: String,
                            arguments: String,
                            activation_type: ActivationType,
                            after_activation_behavior: AfterActivationBehavior,
                            image_uri: Option<String>,
                            hint_inputid: String,
                            hint_buttonStyle: HintButtonStyle,
                            hint_toolTip: String,
                            placement: bool,
                        ) -> Self {
                            Self {
                                content,
                                arguments,
                                activationType: activation_type.into(),
                                afterActivationBehavior: after_activation_behavior.into(),
                                imageUri: image_uri,
                                hint_inputid,
                                hint_buttonStyle: hint_buttonStyle.into(),
                                hint_toolTip,
                                placement,
                            }
                        }
                    }
                    impl ToXML for ActionButton {
                        fn to_xml(&self) -> String {
                            ::alloc::__export::must_use({
                                ::alloc::fmt::format(
                                    format_args!(
                                        "\n          <action content=\"{0}\" arguments=\"{1}\" activationType=\"{2}\" afterActivationBehavior=\"{3}\" imageUri=\"{4}\" hint-inputId=\"{5}\" hint-buttonStyle=\"{6}\" hint-toolTip=\"{7}\" {8} />\n        ",
                                        self.content,
                                        self.arguments,
                                        self.activationType,
                                        self.afterActivationBehavior,
                                        self.imageUri.as_ref().unwrap_or(&"".to_string()),
                                        self.hint_inputid,
                                        self.hint_buttonStyle,
                                        self.hint_toolTip,
                                        if self.placement {
                                            "placement=\"contextMenu\""
                                        } else {
                                            ""
                                        },
                                    ),
                                )
                            })
                        }
                    }
                    /// Learn More Here
                    /// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-action>
                    pub enum ActivationType {
                        #[default]
                        Foreground,
                        Background,
                        Protocol,
                    }
                    #[automatically_derived]
                    impl ::core::default::Default for ActivationType {
                        #[inline]
                        fn default() -> ActivationType {
                            Self::Foreground
                        }
                    }
                    impl Into<String> for ActivationType {
                        fn into(self) -> String {
                            match self {
                                ActivationType::Foreground => "foreground".to_string(),
                                ActivationType::Background => "background".to_string(),
                                ActivationType::Protocol => "protocol".to_string(),
                            }
                        }
                    }
                    /// Learn More Here
                    /// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-action>
                    pub enum AfterActivationBehavior {
                        #[default]
                        Default,
                        PendingUpdate,
                    }
                    #[automatically_derived]
                    impl ::core::default::Default for AfterActivationBehavior {
                        #[inline]
                        fn default() -> AfterActivationBehavior {
                            Self::Default
                        }
                    }
                    impl Into<String> for AfterActivationBehavior {
                        fn into(self) -> String {
                            match self {
                                Self::Default => "default".to_string(),
                                Self::PendingUpdate => "pendingUpdate".to_string(),
                            }
                        }
                    }
                    /// Learn More Here
                    /// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-action>
                    pub enum HintButtonStyle {
                        #[default]
                        None,
                        Success,
                        Critical,
                    }
                    #[automatically_derived]
                    impl ::core::default::Default for HintButtonStyle {
                        #[inline]
                        fn default() -> HintButtonStyle {
                            Self::None
                        }
                    }
                    impl Into<String> for HintButtonStyle {
                        fn into(self) -> String {
                            match self {
                                Self::None => "".to_string(),
                                Self::Success => "Success".to_string(),
                                Self::Critical => "Critical".to_string(),
                            }
                        }
                    }
                    impl ActionElement for ActionButton {}
                    impl ActionableXML for ActionButton {}
                }
                pub mod input {
                    use quick_xml::escape::escape;
                    use crate::{ToXML, map, notification::ActionableXML};
                    use super::ActionElement;
                    #[allow(non_snake_case)]
                    /// Learn more here
                    /// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-input>
                    pub struct Input {
                        id: String,
                        title: String,
                        placeHolder: String,
                        children: String,
                        r#type: String,
                    }
                    /// Learn more here
                    /// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-input>
                    pub enum InputType {
                        Text,
                        Selection(Vec<Selection>),
                    }
                    impl Input {
                        pub fn create_text_input(
                            id: &str,
                            title: &str,
                            place_holder: &str,
                        ) -> Self {
                            unsafe {
                                Self::new_unchecked(
                                    escape(id).into(),
                                    escape(title).into(),
                                    InputType::Text,
                                    escape(place_holder).into(),
                                )
                            }
                        }
                        pub fn create_selection_input(
                            id: &str,
                            title: &str,
                            place_holder: &str,
                            selections: Vec<Selection>,
                        ) -> Self {
                            Self {
                                id: escape(id).into(),
                                title: escape(title).into(),
                                r#type: "selection".into(),
                                placeHolder: escape(place_holder).into(),
                                children: selections
                                    .into_iter()
                                    .map(|x| x.to_xml())
                                    .collect::<Vec<_>>()
                                    .join("\n".into()),
                            }
                        }
                        pub unsafe fn new_unchecked(
                            id: String,
                            title: String,
                            r#type: InputType,
                            place_holder: String,
                        ) -> Self {
                            let (r#type, ch) = match r#type {
                                InputType::Text => ("text", ::alloc::vec::Vec::new()),
                                InputType::Selection(ch) => ("selection", ch),
                            };
                            Self {
                                children: ch
                                    .into_iter()
                                    .map(|x| x.to_xml())
                                    .collect::<Vec<_>>()
                                    .join("\n".into()),
                                id,
                                title,
                                r#type: r#type.into(),
                                placeHolder: place_holder,
                            }
                        }
                        pub fn with_selection(
                            &mut self,
                            children: Vec<Selection>,
                        ) -> &mut Self {
                            self.children = children
                                .into_iter()
                                .map(|x| x.to_xml())
                                .collect::<Vec<_>>()
                                .join("\n".into());
                            self
                        }
                    }
                    impl ActionElement for Input {}
                    impl ToXML for Input {
                        fn to_xml(&self) -> String {
                            ::alloc::__export::must_use({
                                ::alloc::fmt::format(
                                    format_args!(
                                        "\n        <input id=\"{0}\" title=\"{1}\" placeHolderContent=\"{2}\" type=\"{3}\" >\n          {4}\n        </input>\n      ",
                                        self.id,
                                        self.title,
                                        self.placeHolder,
                                        self.r#type,
                                        self.children,
                                    ),
                                )
                            })
                        }
                    }
                    /// Learn more here
                    /// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-input>
                    pub struct Selection {
                        id: String,
                        content: String,
                    }
                    impl Selection {
                        pub fn new(id: &str, content: &str) -> Self {
                            Self {
                                content: escape(content).into(),
                                id: escape(id).into(),
                            }
                        }
                    }
                    impl ToXML for Selection {
                        fn to_xml(&self) -> String {
                            ::alloc::__export::must_use({
                                ::alloc::fmt::format(
                                    format_args!(
                                        "<selection id=\"{0}\" content=\"{1}\" />",
                                        &self.id,
                                        &self.content,
                                    ),
                                )
                            })
                        }
                    }
                    impl ActionableXML for Input {}
                }
                pub use action::ActionButton;
                pub use input::Input;
                pub trait ActionElement {}
            }
            pub mod audio {
                use crate::ToXML;
                /// Learn More About this here
                /// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-audio>
                pub struct Audio {
                    src: String,
                    r#loop: String,
                    silent: String,
                }
                impl Audio {
                    pub fn new(src: Src, r#loop: bool, silent: bool) -> Self {
                        Self {
                            src: src.into(),
                            r#loop: r#loop.to_string(),
                            silent: silent.to_string(),
                        }
                    }
                }
                impl ToXML for Audio {
                    fn to_xml(&self) -> String {
                        ::alloc::__export::must_use({
                            ::alloc::fmt::format(
                                format_args!(
                                    "\n        <audio src=\"{0}\" loop=\"{1}\" silent=\"{2}\" />\n      ",
                                    self.src,
                                    self.r#loop,
                                    self.silent,
                                ),
                            )
                        })
                    }
                }
                /// Learn More About it here
                /// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-audio>
                pub enum Src {
                    #[default]
                    Default,
                    IM,
                    Mail,
                    Reminder,
                    Sms,
                    Alarm,
                    Alarm2,
                    Alarm3,
                    Alarm4,
                    Alarm5,
                    Alarm6,
                    Alarm7,
                    Alarm8,
                    Alarm9,
                    Alarm10,
                    Call,
                    Call2,
                    Call3,
                    Call4,
                    Call5,
                    Call6,
                    Call7,
                    Call8,
                    Call9,
                    Call10,
                }
                #[automatically_derived]
                impl ::core::default::Default for Src {
                    #[inline]
                    fn default() -> Src {
                        Self::Default
                    }
                }
                impl Into<String> for Src {
                    fn into(self) -> String {
                        match self {
                            Self::Default => "ms-winsoundevent:Notification.Default",
                            Self::IM => "ms-winsoundevent:Notification.IM",
                            Self::Mail => "ms-winsoundevent:Notification.Mail",
                            Self::Reminder => "ms-winsoundevent:Notification.Reminder",
                            Self::Sms => "ms-winsoundevent:Notification.Sms",
                            Self::Alarm => "ms-winsoundevent:Notification.Looping.Alarm",
                            Self::Alarm2 => {
                                "ms-winsoundevent:Notification.Looping.Alarm2"
                            }
                            Self::Alarm3 => {
                                "ms-winsoundevent:Notification.Looping.Alarm3"
                            }
                            Self::Alarm4 => {
                                "ms-winsoundevent:Notification.Looping.Alarm4"
                            }
                            Self::Alarm5 => {
                                "ms-winsoundevent:Notification.Looping.Alarm5"
                            }
                            Self::Alarm6 => {
                                "ms-winsoundevent:Notification.Looping.Alarm6"
                            }
                            Self::Alarm7 => {
                                "ms-winsoundevent:Notification.Looping.Alarm7"
                            }
                            Self::Alarm8 => {
                                "ms-winsoundevent:Notification.Looping.Alarm8"
                            }
                            Self::Alarm9 => {
                                "ms-winsoundevent:Notification.Looping.Alarm9"
                            }
                            Self::Alarm10 => {
                                "ms-winsoundevent:Notification.Looping.Alarm10"
                            }
                            Self::Call => "ms-winsoundevent:Notification.Looping.Call",
                            Self::Call2 => "ms-winsoundevent:Notification.Looping.Call2",
                            Self::Call3 => "ms-winsoundevent:Notification.Looping.Call3",
                            Self::Call4 => "ms-winsoundevent:Notification.Looping.Call4",
                            Self::Call5 => "ms-winsoundevent:Notification.Looping.Call5",
                            Self::Call6 => "ms-winsoundevent:Notification.Looping.Call6",
                            Self::Call7 => "ms-winsoundevent:Notification.Looping.Call7",
                            Self::Call8 => "ms-winsoundevent:Notification.Looping.Call8",
                            Self::Call9 => "ms-winsoundevent:Notification.Looping.Call9",
                            Self::Call10 => {
                                "ms-winsoundevent:Notification.Looping.Call10"
                            }
                        }
                            .into()
                    }
                }
            }
            pub mod commands {
                use crate::ToXML;
                /// Learn more about it here
                /// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-commands>
                pub struct Commands {
                    widgets: Vec<Command>,
                }
                impl Commands {
                    pub fn new(commands: Vec<Command>) -> Self {
                        Self { widgets: commands }
                    }
                }
                impl IntoIterator for Commands {
                    type Item = Command;
                    type IntoIter = std::vec::IntoIter<Self::Item>;
                    fn into_iter(self) -> Self::IntoIter {
                        self.widgets.into_iter()
                    }
                }
                /// Learn more about it here
                /// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-command>
                pub struct Command {
                    id: String,
                    arguments: String,
                }
                impl Command {
                    pub fn new(
                        arguments: Option<String>,
                        id: Option<CommandId>,
                    ) -> Self {
                        if let Some(x) = &arguments {
                            if true {
                                if !x.chars().all(|x| x.is_alphanumeric()) {
                                    ::core::panicking::panic(
                                        "assertion failed: x.chars().all(|x| x.is_alphanumeric())",
                                    )
                                }
                            }
                        }
                        Self {
                            id: id
                                .map_or_else(
                                    || "".into(),
                                    |x| ::alloc::__export::must_use({
                                        ::alloc::fmt::format(
                                            format_args!("id=\"{0}\"", Into::<String>::into(x)),
                                        )
                                    }),
                                ),
                            arguments: arguments
                                .map_or_else(
                                    || "".into(),
                                    |x| ::alloc::__export::must_use({
                                        ::alloc::fmt::format(format_args!("arguments=\"{0}\"", x))
                                    }),
                                ),
                        }
                    }
                }
                pub enum CommandId {
                    Snooze,
                    Dismiss,
                    Video,
                    Voice,
                    Decline,
                }
                impl Into<String> for CommandId {
                    fn into(self) -> String {
                        match self {
                            Self::Snooze => "snooze".to_string(),
                            Self::Dismiss => "dismiss".to_string(),
                            Self::Video => "video".to_string(),
                            Self::Voice => "voice".to_string(),
                            Self::Decline => "decline".to_string(),
                        }
                    }
                }
                impl ToXML for Command {
                    fn to_xml(&self) -> String {
                        ::alloc::__export::must_use({
                            ::alloc::fmt::format(
                                format_args!(
                                    "\n        <command {0} {1} />\n      ",
                                    self.arguments,
                                    self.id,
                                ),
                            )
                        })
                    }
                }
            }
            pub mod group {
                mod group {
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
                        pub fn with_subgroup<T: SubgroupXML + 'static>(
                            mut self,
                            subgroup: T,
                        ) -> Self {
                            self.subgroups.push(Box::new(subgroup));
                            self
                        }
                        pub fn new_from(subgroups: Vec<Box<dyn SubgroupXML>>) -> Self {
                            Self { subgroups }
                        }
                    }
                    impl Default for Group {
                        fn default() -> Self {
                            Self {
                                subgroups: ::alloc::vec::Vec::new(),
                            }
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
                            ::alloc::__export::must_use({
                                ::alloc::fmt::format(
                                    format_args!(
                                        "\n      <group>\n        {0}\n      </group>\n    ",
                                        data,
                                    ),
                                )
                            })
                        }
                    }
                }
                mod subgroup {
                    use crate::{notification::visual::TextOrImageElement, ToXML};
                    use super::SubgroupXML;
                    /// Learn More Here
                    /// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-subgroup>
                    pub struct SubGroup {
                        elements: Vec<Box<dyn TextOrImageElement>>,
                        weight: u16,
                        text_stacking: Option<&'static str>,
                    }
                    pub enum AdaptiveSubgroupTextStacking {
                        Default,
                        Top,
                        Center,
                        Bottom,
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
                        pub fn with_visual<T: TextOrImageElement + 'static>(
                            mut self,
                            element: T,
                        ) -> Self {
                            self.elements.push(Box::new(element));
                            self
                        }
                        pub fn with_weight(mut self, weight: u16) -> Self {
                            self.weight = weight;
                            self
                        }
                        pub fn with_text_stacking(
                            mut self,
                            stack: Option<AdaptiveSubgroupTextStacking>,
                        ) -> Self {
                            self.text_stacking = stack.map(|x| x.to_string());
                            self
                        }
                        pub fn new_from(
                            elements: Vec<Box<dyn TextOrImageElement>>,
                        ) -> Self {
                            Self {
                                elements,
                                ..Default::default()
                            }
                        }
                    }
                    impl Default for SubGroup {
                        fn default() -> Self {
                            Self {
                                elements: ::alloc::vec::Vec::new(),
                                text_stacking: None,
                                weight: 0,
                            }
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
                            ::alloc::__export::must_use({
                                ::alloc::fmt::format(
                                    format_args!(
                                        "\n      <subgroup {0} {1}>\n        {2}\n      </subgroup>\n    ",
                                        if self.weight == 0 {
                                            "".to_string()
                                        } else {
                                            ::alloc::__export::must_use({
                                                ::alloc::fmt::format(
                                                    format_args!("hint-weight=\"{0}\"", self.weight),
                                                )
                                            })
                                        },
                                        self
                                            .text_stacking
                                            .map_or_else(
                                                || String::new(),
                                                |s| ::alloc::__export::must_use({
                                                    ::alloc::fmt::format(
                                                        format_args!("hint-textStacking=\"{0}\"", s),
                                                    )
                                                }),
                                            ),
                                        data,
                                    ),
                                )
                            })
                        }
                    }
                }
                pub use group::*;
                pub use subgroup::*;
                use crate::ToXML;
                pub trait SubgroupXML: ToXML {}
            }
            pub mod header {
                use quick_xml::escape::escape;
                use crate::ToXML;
                /// Learn more about it here
                /// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-header>
                pub struct Header {
                    id: String,
                    title: String,
                    arguments: String,
                    activation_type: String,
                }
                impl Header {
                    pub fn new(
                        id: &str,
                        title: &str,
                        arguments: &str,
                        activation_type: Option<HeaderActivationType>,
                    ) -> Self {
                        Self {
                            id: escape(id).into(),
                            title: escape(title).into(),
                            arguments: escape(arguments).into(),
                            activation_type: activation_type.unwrap_or_default().into(),
                        }
                    }
                }
                impl ToXML for Header {
                    fn to_xml(&self) -> String {
                        ::alloc::__export::must_use({
                            ::alloc::fmt::format(
                                format_args!(
                                    "\n      <header title=\\\"{0}\\\" arguments=\\\"{1}\\\" id=\\\"{2}\\\" activationType=\"{3}\" />\n    ",
                                    self.title,
                                    self.arguments,
                                    self.id,
                                    self.activation_type,
                                ),
                            )
                        })
                    }
                }
                /// Learn more about it here
                /// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-header>
                pub enum HeaderActivationType {
                    #[default]
                    Foreground,
                    Protocol,
                }
                #[automatically_derived]
                impl ::core::default::Default for HeaderActivationType {
                    #[inline]
                    fn default() -> HeaderActivationType {
                        Self::Foreground
                    }
                }
                impl Into<String> for HeaderActivationType {
                    fn into(self) -> String {
                        match self {
                            HeaderActivationType::Foreground => "foreground".to_string(),
                            HeaderActivationType::Protocol => "protocol".to_string(),
                        }
                    }
                }
            }
            pub mod visual {
                pub mod image {
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
                                Placement::AppLogoOverride => {
                                    "placement=\"appLogoOverride\"".to_string()
                                }
                                Placement::Hero => "placement=\"hero\"".to_string(),
                                Placement::None => "".to_string(),
                            }
                        }
                    }
                    pub enum ImageCrop {
                        #[default]
                        Default,
                        None,
                        Circle,
                    }
                    #[automatically_derived]
                    impl ::core::fmt::Debug for ImageCrop {
                        #[inline]
                        fn fmt(
                            &self,
                            f: &mut ::core::fmt::Formatter,
                        ) -> ::core::fmt::Result {
                            ::core::fmt::Formatter::write_str(
                                f,
                                match self {
                                    ImageCrop::Default => "Default",
                                    ImageCrop::None => "None",
                                    ImageCrop::Circle => "Circle",
                                },
                            )
                        }
                    }
                    #[automatically_derived]
                    impl ::core::clone::Clone for ImageCrop {
                        #[inline]
                        fn clone(&self) -> ImageCrop {
                            match self {
                                ImageCrop::Default => ImageCrop::Default,
                                ImageCrop::None => ImageCrop::None,
                                ImageCrop::Circle => ImageCrop::Circle,
                            }
                        }
                    }
                    #[automatically_derived]
                    impl ::core::default::Default for ImageCrop {
                        #[inline]
                        fn default() -> ImageCrop {
                            Self::Default
                        }
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
                    pub enum AdaptiveImageAlign {
                        #[default]
                        Default,
                        Stretch,
                        Left,
                        Center,
                        Right,
                    }
                    #[automatically_derived]
                    impl ::core::fmt::Debug for AdaptiveImageAlign {
                        #[inline]
                        fn fmt(
                            &self,
                            f: &mut ::core::fmt::Formatter,
                        ) -> ::core::fmt::Result {
                            ::core::fmt::Formatter::write_str(
                                f,
                                match self {
                                    AdaptiveImageAlign::Default => "Default",
                                    AdaptiveImageAlign::Stretch => "Stretch",
                                    AdaptiveImageAlign::Left => "Left",
                                    AdaptiveImageAlign::Center => "Center",
                                    AdaptiveImageAlign::Right => "Right",
                                },
                            )
                        }
                    }
                    #[automatically_derived]
                    impl ::core::clone::Clone for AdaptiveImageAlign {
                        #[inline]
                        fn clone(&self) -> AdaptiveImageAlign {
                            match self {
                                AdaptiveImageAlign::Default => AdaptiveImageAlign::Default,
                                AdaptiveImageAlign::Stretch => AdaptiveImageAlign::Stretch,
                                AdaptiveImageAlign::Left => AdaptiveImageAlign::Left,
                                AdaptiveImageAlign::Center => AdaptiveImageAlign::Center,
                                AdaptiveImageAlign::Right => AdaptiveImageAlign::Right,
                            }
                        }
                    }
                    #[automatically_derived]
                    impl ::core::default::Default for AdaptiveImageAlign {
                        #[inline]
                        fn default() -> AdaptiveImageAlign {
                            Self::Default
                        }
                    }
                    impl ToString for AdaptiveImageAlign {
                        fn to_string(&self) -> String {
                            match self {
                                AdaptiveImageAlign::Default => "".to_string(),
                                AdaptiveImageAlign::Stretch => {
                                    "hint-align=\"stretch\"".to_string()
                                }
                                AdaptiveImageAlign::Left => {
                                    "hint-align=\"left\"".to_string()
                                }
                                AdaptiveImageAlign::Center => {
                                    "hint-align=\"center\"".to_string()
                                }
                                AdaptiveImageAlign::Right => {
                                    "hint-align=\"right\"".to_string()
                                }
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
                            return ::alloc::__export::must_use({
                                ::alloc::fmt::format(format_args!("file:///{0}", src))
                            });
                        }
                        src
                    }
                    impl Image {
                        /// The `src` should be the either of the following following
                        /// - `https://url or http://url`
                        /// - `file:///path/to/file`
                        ///
                        /// If none of the above is provided, the `src` will be set to `file:///path/to/file`
                        pub fn create<T: Into<String>>(id: u64, src: T) -> Self {
                            Self::new(
                                id,
                                src.into(),
                                None,
                                false,
                                Placement::None,
                                ImageCrop::Default,
                                false,
                            )
                        }
                        /// The `src` should be the either of the following following
                        /// - `https://url or http://url`
                        /// - `file:///path/to/file`
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
                            ::alloc::__export::must_use({
                                ::alloc::fmt::format(
                                    format_args!(
                                        "\n        <image id=\"{2}\" {1} {0} src=\"{3}\" {5} {4} {6} {7} />\n      ",
                                        self.align.to_string(),
                                        match self.no_margin {
                                            true => "hint-remove-margin=\"true\"".to_string(),
                                            false => "".to_string(),
                                        },
                                        self.id,
                                        ::alloc::__export::must_use({
                                                ::alloc::fmt::format(format_args!("{0}", self.src))
                                            })
                                            .replace("\\\\", "\\"),
                                        self
                                            .alt
                                            .clone()
                                            .map_or_else(
                                                || ::alloc::__export::must_use({
                                                    ::alloc::fmt::format(format_args!(""))
                                                }),
                                                |x| ::alloc::__export::must_use({
                                                    ::alloc::fmt::format(format_args!("alt=\"{0}\"", x))
                                                }),
                                            ),
                                        if self.add_image_query {
                                            "addImageQuery=\"True\""
                                        } else {
                                            ""
                                        },
                                        self.placement.to_string(),
                                        self.crop.to_string(),
                                    ),
                                )
                            })
                        }
                    }
                }
                pub mod progress {
                    use crate::{
                        notification::{AdaptiveText, ToastVisualableXML},
                        ToXML,
                    };
                    use super::VisualElement;
                    #[allow(non_snake_case)]
                    /// Learn more here
                    /// <https://learn.microsoft.com/en-us/uwp/schemas/tiles/toastschema/element-progress>
                    pub struct Progress {
                        title: Option<String>,
                        value_string_override: Option<String>,
                        status: String,
                        value: String,
                    }
                    pub enum ProgressValue<'a> {
                        Percentage(f64),
                        BindTo(&'a str),
                        Indeterminate,
                    }
                    impl<'a> ToString for ProgressValue<'a> {
                        fn to_string(&self) -> String {
                            match self {
                                ProgressValue::Percentage(x) => {
                                    ::alloc::__export::must_use({
                                        ::alloc::fmt::format(format_args!("{0}", x / 100.0))
                                    })
                                }
                                ProgressValue::BindTo(x) => {
                                    if true {
                                        if !x.chars().all(|x| x.is_alphabetic()) {
                                            ::core::panicking::panic(
                                                "assertion failed: x.chars().all(|x| x.is_alphabetic())",
                                            )
                                        }
                                    }
                                    ::alloc::__export::must_use({
                                        ::alloc::fmt::format(format_args!("{{{0}}}", x))
                                    })
                                }
                                ProgressValue::Indeterminate => "indeterminate".to_string(),
                            }
                        }
                    }
                    impl Progress {
                        pub fn create(
                            status_text: AdaptiveText,
                            value: ProgressValue,
                        ) -> Self {
                            unsafe {
                                Self::new_unchecked(
                                    None,
                                    status_text.to_string(),
                                    value,
                                    None,
                                )
                            }
                        }
                        pub fn with_title<T: AsRef<str>>(
                            mut self,
                            title: AdaptiveText,
                        ) -> Self {
                            self.title = Some(title.to_string());
                            self
                        }
                        pub fn with_value(mut self, value: ProgressValue) -> Self {
                            self.value = value.to_string();
                            self
                        }
                        pub fn override_value_string(
                            mut self,
                            value: AdaptiveText,
                        ) -> Self {
                            self.value_string_override = Some(value.to_string());
                            self
                        }
                        pub unsafe fn new_unchecked(
                            title: Option<String>,
                            status_text: String,
                            value: ProgressValue,
                            value_string_override: Option<String>,
                        ) -> Self {
                            Self {
                                title,
                                status: status_text,
                                value: value.to_string(),
                                value_string_override,
                            }
                        }
                    }
                    impl VisualElement for Progress {}
                    impl ToastVisualableXML for Progress {}
                    impl ToXML for Progress {
                        fn to_xml(&self) -> String {
                            ::alloc::__export::must_use({
                                ::alloc::fmt::format(
                                    format_args!(
                                        "\n        <progress {0} status=\"{1}\" value=\"{2}\" {3} />\n      ",
                                        self
                                            .title
                                            .clone()
                                            .map_or_else(
                                                || ::alloc::__export::must_use({
                                                    ::alloc::fmt::format(format_args!(""))
                                                }),
                                                |x| ::alloc::__export::must_use({
                                                    ::alloc::fmt::format(format_args!("title=\"{0}\"", x))
                                                }),
                                            ),
                                        self.status,
                                        self.value,
                                        self
                                            .value_string_override
                                            .clone()
                                            .map_or_else(
                                                || ::alloc::__export::must_use({
                                                    ::alloc::fmt::format(format_args!(""))
                                                }),
                                                |x| ::alloc::__export::must_use({
                                                    ::alloc::fmt::format(
                                                        format_args!("valueStringOverride=\"{0}\"", x),
                                                    )
                                                }),
                                            ),
                                    ),
                                )
                            })
                        }
                    }
                }
                pub mod text {
                    use quick_xml::escape::escape;
                    use crate::{notification::ToastVisualableXML, ToXML};
                    use super::{TextOrImageElement, VisualElement};
                    pub struct AttributionPlacement;
                    #[automatically_derived]
                    impl ::core::fmt::Debug for AttributionPlacement {
                        #[inline]
                        fn fmt(
                            &self,
                            f: &mut ::core::fmt::Formatter,
                        ) -> ::core::fmt::Result {
                            ::core::fmt::Formatter::write_str(f, "AttributionPlacement")
                        }
                    }
                    #[automatically_derived]
                    impl ::core::clone::Clone for AttributionPlacement {
                        #[inline]
                        fn clone(&self) -> AttributionPlacement {
                            *self
                        }
                    }
                    #[automatically_derived]
                    impl ::core::marker::Copy for AttributionPlacement {}
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
                    #[automatically_derived]
                    impl ::core::fmt::Debug for HintStyle {
                        #[inline]
                        fn fmt(
                            &self,
                            f: &mut ::core::fmt::Formatter,
                        ) -> ::core::fmt::Result {
                            ::core::fmt::Formatter::write_str(
                                f,
                                match self {
                                    HintStyle::Default => "Default",
                                    HintStyle::Caption => "Caption",
                                    HintStyle::CaptionSubtle => "CaptionSubtle",
                                    HintStyle::Body => "Body",
                                    HintStyle::BodySubtle => "BodySubtle",
                                    HintStyle::Base => "Base",
                                    HintStyle::BaseSubtle => "BaseSubtle",
                                    HintStyle::Subtitle => "Subtitle",
                                    HintStyle::SubtitleSubtle => "SubtitleSubtle",
                                    HintStyle::Title => "Title",
                                    HintStyle::TitleSubtle => "TitleSubtle",
                                    HintStyle::TitleNumeral => "TitleNumeral",
                                    HintStyle::Subheader => "Subheader",
                                    HintStyle::SubheaderSubtle => "SubheaderSubtle",
                                    HintStyle::SubheaderNumeral => "SubheaderNumeral",
                                    HintStyle::Header => "Header",
                                    HintStyle::HeaderSubtle => "HeaderSubtle",
                                    HintStyle::HeaderNumeral => "HeaderNumeral",
                                },
                            )
                        }
                    }
                    #[automatically_derived]
                    impl ::core::clone::Clone for HintStyle {
                        #[inline]
                        fn clone(&self) -> HintStyle {
                            *self
                        }
                    }
                    #[automatically_derived]
                    impl ::core::marker::Copy for HintStyle {}
                    #[automatically_derived]
                    impl ::core::default::Default for HintStyle {
                        #[inline]
                        fn default() -> HintStyle {
                            Self::Default
                        }
                    }
                    impl ToString for HintStyle {
                        fn to_string(&self) -> String {
                            match self {
                                HintStyle::Base => r#"hint-style="base""#.to_string(),
                                HintStyle::Title => r#"hint-style="title""#.to_string(),
                                HintStyle::Subtitle => {
                                    r#"hint-style="subtitle""#.to_string()
                                }
                                HintStyle::CaptionSubtle => {
                                    r#"hint-style="captionSubtle""#.to_string()
                                }
                                HintStyle::BaseSubtle => {
                                    r#"hint-style="baseSubtle""#.to_string()
                                }
                                HintStyle::TitleSubtle => {
                                    r#"hint-style="titleSubtle""#.to_string()
                                }
                                HintStyle::SubtitleSubtle => {
                                    r#"hint-style="subtitleSubtle""#.to_string()
                                }
                                HintStyle::Caption => r#"hint-style="caption""#.to_string(),
                                HintStyle::Body => r#"hint-style="body""#.to_string(),
                                HintStyle::BodySubtle => {
                                    r#"hint-style="bodySubtle""#.to_string()
                                }
                                HintStyle::Subheader => {
                                    r#"hint-style="subheader""#.to_string()
                                }
                                HintStyle::SubheaderSubtle => {
                                    r#"hint-style="subheaderSubtle""#.to_string()
                                }
                                HintStyle::SubheaderNumeral => {
                                    r#"hint-style="subheaderNumeral""#.to_string()
                                }
                                HintStyle::Header => r#"hint-style="header""#.to_string(),
                                HintStyle::HeaderSubtle => {
                                    r#"hint-style="headerSubtle""#.to_string()
                                }
                                HintStyle::HeaderNumeral => {
                                    r#"hint-style="headerNumeral""#.to_string()
                                }
                                HintStyle::Default => "".to_string(),
                                HintStyle::TitleNumeral => {
                                    "hint-style=\"titleNumeral\"".to_string()
                                }
                            }
                        }
                    }
                    pub enum HintAlign {
                        Right,
                        Auto,
                        Left,
                        Center,
                        #[default]
                        Default,
                    }
                    #[automatically_derived]
                    impl ::core::fmt::Debug for HintAlign {
                        #[inline]
                        fn fmt(
                            &self,
                            f: &mut ::core::fmt::Formatter,
                        ) -> ::core::fmt::Result {
                            ::core::fmt::Formatter::write_str(
                                f,
                                match self {
                                    HintAlign::Right => "Right",
                                    HintAlign::Auto => "Auto",
                                    HintAlign::Left => "Left",
                                    HintAlign::Center => "Center",
                                    HintAlign::Default => "Default",
                                },
                            )
                        }
                    }
                    #[automatically_derived]
                    impl ::core::clone::Clone for HintAlign {
                        #[inline]
                        fn clone(&self) -> HintAlign {
                            *self
                        }
                    }
                    #[automatically_derived]
                    impl ::core::marker::Copy for HintAlign {}
                    #[automatically_derived]
                    impl ::core::default::Default for HintAlign {
                        #[inline]
                        fn default() -> HintAlign {
                            Self::Default
                        }
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
                    #[automatically_derived]
                    #[allow(non_snake_case)]
                    impl ::core::default::Default for Text {
                        #[inline]
                        fn default() -> Text {
                            Text {
                                body: ::core::default::Default::default(),
                                id: ::core::default::Default::default(),
                                lang: ::core::default::Default::default(),
                                placement: ::core::default::Default::default(),
                                style: ::core::default::Default::default(),
                                align: ::core::default::Default::default(),
                                wrap: ::core::default::Default::default(),
                                callScenarioCenterAlign: ::core::default::Default::default(),
                                maxLines: ::core::default::Default::default(),
                                minLines: ::core::default::Default::default(),
                            }
                        }
                    }
                    impl TextOrImageElement for Text {}
                    impl Text {
                        pub fn create(id: u64, body: &str) -> Self {
                            unsafe {
                                Self::new_unchecked(
                                    id,
                                    None,
                                    None,
                                    escape(body).to_string(),
                                )
                            }
                        }
                        pub fn create_binded(id: u64, binds: &str) -> Self {
                            if true {
                                if !binds.chars().all(|x| x.is_alphabetic()) {
                                    ::core::panicking::panic(
                                        "assertion failed: binds.chars().all(|x| x.is_alphabetic())",
                                    )
                                }
                            }
                            unsafe {
                                Self::new_unchecked(
                                    id,
                                    None,
                                    None,
                                    ::alloc::__export::must_use({
                                        ::alloc::fmt::format(format_args!("{{{0}}}", binds))
                                    }),
                                )
                            }
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
                        pub fn with_placement(
                            mut self,
                            placement: AttributionPlacement,
                        ) -> Self {
                            self.placement = Some(placement);
                            self
                        }
                        /// Only for IncomingCall scenarios
                        pub fn align_center(mut self, center: bool) -> Self {
                            self.callScenarioCenterAlign = center;
                            self
                        }
                        pub fn wrap(mut self, wrap: bool) -> Self {
                            self.wrap = wrap;
                            self
                        }
                        pub fn max_lines(mut self, max_lines: u32) -> Self {
                            self.maxLines = max_lines;
                            self
                        }
                        pub fn min_lines(mut self, min_lines: u32) -> Self {
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
                            ::alloc::__export::must_use({
                                ::alloc::fmt::format(
                                    format_args!(
                                        "\n        <text id=\"{0}\" {1} {2} {3} {4} {5} {6} {7} {8}>\n          {9}\n        </text>\n      ",
                                        self.id,
                                        if self.wrap { "hint-wrap='true'" } else { "" },
                                        if self.maxLines > 0 {
                                            ::alloc::__export::must_use({
                                                ::alloc::fmt::format(
                                                    format_args!("hint-maxLines=\'{0}\'", self.maxLines),
                                                )
                                            })
                                        } else {
                                            "".to_string()
                                        },
                                        if self.minLines > 0 {
                                            ::alloc::__export::must_use({
                                                ::alloc::fmt::format(
                                                    format_args!("hint-minLines=\'{0}\'", self.minLines),
                                                )
                                            })
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
                                            .map_or_else(
                                                || ::alloc::__export::must_use({
                                                    ::alloc::fmt::format(format_args!(""))
                                                }),
                                                |x| ::alloc::__export::must_use({
                                                    ::alloc::fmt::format(format_args!("lang=\"{0}\"", x))
                                                }),
                                            ),
                                        self
                                            .placement
                                            .map_or_else(|| "", |_| "placement=\"attribution\""),
                                        self.body,
                                    ),
                                )
                            })
                        }
                    }
                }
                use crate::ToXML;
                pub trait VisualElement {}
                pub trait TextOrImageElement: VisualElement + ToXML {}
                pub use image::{Image, Placement};
                pub use progress::Progress;
                pub use text::{AttributionPlacement, Text};
            }
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
                            if true {
                                if !x.chars().all(|x| x.is_alphabetic()) {
                                    ::core::panicking::panic(
                                        "assertion failed: x.chars().all(|x| x.is_alphabetic())",
                                    )
                                }
                            }
                            ::alloc::__export::must_use({
                                ::alloc::fmt::format(format_args!("{{{0}}}", x))
                            })
                        }
                    }
                }
            }
        }
        pub use widgets::*;
        /// This is a partial version of notification
        /// You can convert it to a Notification **but it will lost the handler tokens**
        ///
        /// We have to call [`OwnedPartialNotification::get_partial`] to get the PartialNotification object
        /// to work on it
        pub struct OwnedPartialNotification {
            pub(crate) notif: ToastNotification,
        }
        impl OwnedPartialNotification {
            pub fn get_partial<'a>(&'a self) -> PartialNotification<'a> {
                PartialNotification {
                    _toast: &self.notif,
                }
            }
        }
        /// This is a partial version of notification
        /// You can convert it to a Notification **but it will lost the handler tokens**
        pub struct PartialNotification<'a> {
            pub(crate) _toast: &'a ToastNotification,
        }
        impl<'a> PartialNotification<'a> {
            #[deprecated = "Use `upgrade` instead"]
            pub fn cast(self, notifier: &'a ToastsNotifier) -> Notification<'a> {
                self.upgrade(notifier)
            }
            /// Converts to a Notification **but it will lost the handler tokens**
            pub fn upgrade(self, notifier: &'a ToastsNotifier) -> Notification<'a> {
                Notification {
                    _toast: self._toast.clone(),
                    _notifier: notifier,
                    activated_event_handler_token: None,
                    dismissed_event_handler_token: None,
                    failed_event_handler_token: None,
                }
            }
        }
        impl<'a> NotificationImpl for PartialNotification<'a> {
            fn notif(&self) -> &ToastNotification {
                &self._toast
            }
        }
        impl<'a> Notification<'a> {
            pub fn show(&self) -> Result<(), NotifError> {
                Ok(self._notifier.get_raw_handle().Show(&self._toast)?)
            }
            pub unsafe fn as_raw(&self) -> &ToastNotification {
                &self._toast
            }
        }
        /// The Notification Object
        pub struct Notification<'a> {
            pub(crate) _toast: ToastNotification,
            pub(crate) _notifier: &'a ToastsNotifier,
            pub activated_event_handler_token: Option<i64>,
            pub dismissed_event_handler_token: Option<i64>,
            pub failed_event_handler_token: Option<i64>,
        }
        impl NotificationImpl for Notification<'_> {
            fn notif(&self) -> &ToastNotification {
                &self._toast
            }
        }
        pub enum ToastDuration {
            None,
            Long,
            Short,
        }
        pub enum Scenario {
            Default,
            Reminder,
            Alarm,
            IncomingCall,
            Urgent,
        }
        pub trait ActionableXML: ActionElement + ToXML {}
        pub trait ToastVisualableXML: VisualElement + ToXML {}
        /// The way to build a Notification
        pub struct NotificationBuilder {
            audio: Option<Audio>,
            header: Option<Header>,
            commands: Option<Commands>,
            expiry: Option<Duration>,
            visual: Vec<Box<dyn ToastVisualableXML>>,
            actions: Vec<Box<dyn ActionableXML>>,
            on_activated: Option<NotificationActivatedEventHandler>,
            on_failed: Option<NotificationFailedEventHandler>,
            on_dismissed: Option<NotificationDismissedEventHandler>,
            duration: &'static str,
            scenario: &'static str,
            use_button_style: &'static str,
            pub values: HashMap<String, String>,
        }
        impl NotificationBuilder {
            pub fn new() -> Self {
                Self {
                    visual: ::alloc::vec::Vec::new(),
                    actions: ::alloc::vec::Vec::new(),
                    audio: None,
                    commands: None,
                    header: None,
                    expiry: None,
                    on_activated: None,
                    on_dismissed: None,
                    on_failed: None,
                    duration: "",
                    scenario: "",
                    use_button_style: "",
                    values: HashMap::new(),
                }
            }
            pub fn audio(mut self, audio: Audio) -> Self {
                self.audio = Some(audio);
                self
            }
            pub fn header(mut self, header: Header) -> Self {
                self.header = Some(header);
                self
            }
            pub fn commands(mut self, commands: Commands) -> Self {
                self.commands = Some(commands);
                self
            }
            pub fn with_duration(mut self, duration: ToastDuration) -> Self {
                match duration {
                    ToastDuration::None => self.duration = "",
                    ToastDuration::Short => self.duration = "duration=\"short\"",
                    ToastDuration::Long => self.duration = "duration=\"long\"",
                }
                self
            }
            /// Sets the ExpirationTime of the notification
            ///
            /// Please note that its accurate upto **seconds only**
            ///
            /// ## Example
            /// ```rust
            /// fn main() {
            ///   let builder = NotificationBuilder::new()
            ///     .with_expiry(Duration::from_secs(30));
            /// }
            /// ```
            pub fn with_expiry(mut self, expiry: Duration) -> Self {
                self.expiry = Some(expiry);
                self
            }
            pub fn with_scenario(mut self, scenario: Scenario) -> Self {
                match scenario {
                    Scenario::Default => self.scenario = "",
                    Scenario::Alarm => self.scenario = "scenario=\"alarm\"",
                    Scenario::Reminder => self.scenario = "scenario=\"reminder\"",
                    Scenario::IncomingCall => self.scenario = "scenario=\"incomingCall\"",
                    Scenario::Urgent => self.scenario = "scenario=\"urgent\"",
                }
                self
            }
            pub fn with_use_button_style(mut self, use_button_style: bool) -> Self {
                if use_button_style {
                    self.use_button_style = "useButtonStyle=\"True\""
                } else {
                    self.use_button_style = ""
                }
                self
            }
            pub fn value<T: Into<String>, E: Into<String>>(
                mut self,
                key: T,
                value: E,
            ) -> Self {
                self.values.insert(key.into(), value.into());
                self
            }
            pub fn values(mut self, values: HashMap<String, String>) -> Self {
                self.values = values;
                self
            }
            pub fn action<T: ActionableXML + 'static>(mut self, action: T) -> Self {
                self.actions.push(Box::new(action));
                self
            }
            pub fn actions(mut self, actions: Vec<Box<dyn ActionableXML>>) -> Self {
                self.actions = actions;
                self
            }
            pub fn visual<T: ToastVisualableXML + 'static>(mut self, visual: T) -> Self {
                self.visual.push(Box::new(visual));
                self
            }
            pub fn visuals(mut self, visual: Vec<Box<dyn ToastVisualableXML>>) -> Self {
                self.visual = visual;
                self
            }
            pub fn on_activated(
                mut self,
                on_activated: NotificationActivatedEventHandler,
            ) -> Self {
                self.on_activated = Some(on_activated);
                self
            }
            pub fn on_failed(
                mut self,
                on_failed: NotificationFailedEventHandler,
            ) -> Self {
                self.on_failed = Some(on_failed);
                self
            }
            pub fn on_dismissed(
                mut self,
                on_dismissed: NotificationDismissedEventHandler,
            ) -> Self {
                self.on_dismissed = Some(on_dismissed);
                self
            }
            pub fn build<'a>(
                self,
                sequence: u32,
                _notifier: &'a ToastsNotifier,
                tag: &str,
                group: &str,
            ) -> Result<Notification<'a>, NotifError> {
                let visual = self
                    .visual
                    .into_iter()
                    .map(|x| x.to_xml())
                    .collect::<Vec<_>>()
                    .join("\n".into());
                let actions = self
                    .actions
                    .into_iter()
                    .map(|x| x.to_xml())
                    .collect::<Vec<_>>()
                    .join("\n".into());
                let audio = self.audio.map_or_else(|| "".into(), |x| x.to_xml());
                let header = self.header.map_or_else(|| "".into(), |x| x.to_xml());
                let commands = self
                    .commands
                    .map_or_else(
                        || "".into(),
                        |x| {
                            ::alloc::__export::must_use({
                                ::alloc::fmt::format(
                                    format_args!(
                                        "\n        <commands>\n          {0}\n        </commands>\n      ",
                                        x
                                            .into_iter()
                                            .map(|x| x.to_xml())
                                            .collect::<Vec<_>>()
                                            .join("\n".into()),
                                    ),
                                )
                            })
                        },
                    );
                let _xml = ::alloc::__export::must_use({
                    ::alloc::fmt::format(
                        format_args!(
                            "\n      <toast {0} {1} {2}>\n        {3}\n        {4}\n        {5}\n        <visual>\n          <binding template=\'ToastGeneric\'>\n            {6}\n          </binding>\n        </visual>\n        <actions>\n          {7}\n        </actions>\n      </toast>\n    ",
                            self.duration,
                            self.scenario,
                            self.use_button_style,
                            audio,
                            commands,
                            header,
                            visual,
                            actions,
                        ),
                    )
                });
                let doc = XmlDocument::new()?;
                doc.LoadXml(&HSTRING::from(_xml))?;
                let data = NotificationData::new()?;
                data.SetSequenceNumber(sequence)?;
                for (key, value) in self.values {
                    data.Values()?.Insert(&key.into(), &value.into())?;
                }
                let mut activated_event_handler_token = None;
                let mut dismissed_event_handler_token = None;
                let mut failed_event_handler_token = None;
                let toast = ToastNotification::CreateToastNotification(&doc)?;
                if let Some(x) = self.on_activated {
                    let token = toast.Activated(&x.handler)?;
                    activated_event_handler_token = Some(token);
                }
                if let Some(x) = self.on_dismissed {
                    let token = toast.Dismissed(&x.handler)?;
                    dismissed_event_handler_token = Some(token);
                }
                if let Some(x) = self.on_failed {
                    let token = toast.Failed(&x.handler)?;
                    failed_event_handler_token = Some(token);
                }
                if let Some(x) = self.expiry {
                    let calendar = Calendar::new()?;
                    if x.as_secs() > i32::MAX as u64 {
                        return Err(NotifError::DurationTooLong);
                    }
                    calendar.AddSeconds(x.as_secs() as i32)?;
                    let dt = calendar.GetDateTime()?;
                    toast
                        .SetExpirationTime(
                            &PropertyValue::CreateDateTime(dt)?
                                .cast::<IReference<DateTime>>()?,
                        )?;
                }
                toast.SetTag(&tag.into())?;
                toast.SetGroup(&group.into())?;
                toast.SetData(&data)?;
                Ok(Notification {
                    _toast: toast,
                    _notifier,
                    activated_event_handler_token,
                    dismissed_event_handler_token,
                    failed_event_handler_token,
                })
            }
        }
    }
    pub mod notifier {
        use std::{sync::Arc, thread};
        use windows::{
            core::HSTRING,
            Win32::{
                System::Com::{
                    CoInitializeEx, CoRegisterClassObject, CLSCTX_LOCAL_SERVER,
                    COINIT_APARTMENTTHREADED, REGCLS_MULTIPLEUSE,
                },
                UI::{
                    Shell::SetCurrentProcessExplicitAppUserModelID,
                    WindowsAndMessaging::{
                        DispatchMessageW, GetMessageW, TranslateMessage, MSG,
                    },
                },
            },
            UI::Notifications::{
                NotificationData, NotificationUpdateResult, ToastNotificationHistory,
                ToastNotificationManager, ToastNotifier,
            },
        };
        use windows_core::{IUnknown, GUID};
        use crate::{
            notification::OwnedPartialNotification,
            notifier::activator::ToastActivationManager, NotifError,
        };
        use super::NotificationDataSet;
        mod activator {
            use windows::core::implement;
            use windows::Win32::UI::Notifications::{
                INotificationActivationCallback, INotificationActivationCallback_Impl,
                NOTIFICATION_USER_INPUT_DATA,
            };
            pub struct ToastActivationManager;
            impl ToastActivationManager {
                #[inline(always)]
                const fn into_outer(self) -> ToastActivationManager_Impl {
                    ToastActivationManager_Impl {
                        identity: &ToastActivationManager_Impl::VTABLE_IDENTITY,
                        interface1_inotificationactivat: &ToastActivationManager_Impl::VTABLE_INTERFACE1_INOTIFICATIONACTIVAT,
                        count: ::windows_core::imp::WeakRefCount::new(),
                        this: self,
                    }
                }
                /// This converts a partially-constructed COM object (in the sense that it contains
                /// application state but does not yet have vtable and reference count constructed)
                /// into a `StaticComObject`. This allows the COM object to be stored in static
                /// (global) variables.
                pub const fn into_static(self) -> ::windows_core::StaticComObject<Self> {
                    ::windows_core::StaticComObject::from_outer(self.into_outer())
                }
            }
            #[repr(C)]
            #[allow(non_camel_case_types)]
            pub struct ToastActivationManager_Impl {
                identity: &'static ::windows_core::IInspectable_Vtbl,
                interface1_inotificationactivat: &'static <INotificationActivationCallback as ::windows_core::Interface>::Vtable,
                this: ToastActivationManager,
                count: ::windows_core::imp::WeakRefCount,
            }
            impl ::core::ops::Deref for ToastActivationManager_Impl {
                type Target = ToastActivationManager;
                #[inline(always)]
                fn deref(&self) -> &Self::Target {
                    &self.this
                }
            }
            impl ToastActivationManager_Impl {
                const VTABLE_IDENTITY: ::windows_core::IInspectable_Vtbl = ::windows_core::IInspectable_Vtbl::new::<
                    ToastActivationManager_Impl,
                    INotificationActivationCallback,
                    0,
                >();
                const VTABLE_INTERFACE1_INOTIFICATIONACTIVAT: <INotificationActivationCallback as ::windows_core::Interface>::Vtable = <INotificationActivationCallback as ::windows_core::Interface>::Vtable::new::<
                    ToastActivationManager_Impl,
                    -1isize,
                >();
            }
            impl ::windows_core::IUnknownImpl for ToastActivationManager_Impl {
                type Impl = ToastActivationManager;
                #[inline(always)]
                fn get_impl(&self) -> &Self::Impl {
                    &self.this
                }
                #[inline(always)]
                fn get_impl_mut(&mut self) -> &mut Self::Impl {
                    &mut self.this
                }
                #[inline(always)]
                fn into_inner(self) -> Self::Impl {
                    self.this
                }
                #[inline(always)]
                fn AddRef(&self) -> u32 {
                    self.count.add_ref()
                }
                #[inline(always)]
                unsafe fn Release(self_: *mut Self) -> u32 {
                    let remaining = (*self_).count.release();
                    if remaining == 0 {
                        _ = ::windows_core::imp::Box::from_raw(self_);
                    }
                    remaining
                }
                #[inline(always)]
                fn is_reference_count_one(&self) -> bool {
                    self.count.is_one()
                }
                unsafe fn GetTrustLevel(
                    &self,
                    value: *mut i32,
                ) -> ::windows_core::HRESULT {
                    if value.is_null() {
                        return ::windows_core::imp::E_POINTER;
                    }
                    *value = 0;
                    ::windows_core::HRESULT(0)
                }
                fn to_object(&self) -> ::windows_core::ComObject<Self::Impl> {
                    self.count.add_ref();
                    unsafe {
                        ::windows_core::ComObject::from_raw(
                            ::core::ptr::NonNull::new_unchecked(
                                self as *const Self as *mut Self,
                            ),
                        )
                    }
                }
                unsafe fn QueryInterface(
                    &self,
                    iid: *const ::windows_core::GUID,
                    interface: *mut *mut ::core::ffi::c_void,
                ) -> ::windows_core::HRESULT {
                    unsafe {
                        if iid.is_null() || interface.is_null() {
                            return ::windows_core::imp::E_POINTER;
                        }
                        let iid = *iid;
                        let interface_ptr: *const ::core::ffi::c_void = 'found: {
                            if iid
                                == <::windows_core::IUnknown as ::windows_core::Interface>::IID
                                || iid
                                    == <::windows_core::IInspectable as ::windows_core::Interface>::IID
                                || iid
                                    == <::windows_core::imp::IAgileObject as ::windows_core::Interface>::IID
                            {
                                break 'found &self.identity as *const _
                                    as *const ::core::ffi::c_void;
                            }
                            if <INotificationActivationCallback as ::windows_core::Interface>::Vtable::matches(
                                &iid,
                            ) {
                                break 'found &self.interface1_inotificationactivat
                                    as *const _ as *const ::core::ffi::c_void;
                            }
                            if iid
                                == <::windows_core::imp::IMarshal as ::windows_core::Interface>::IID
                            {
                                return ::windows_core::imp::marshaler(
                                    self.to_interface(),
                                    interface,
                                );
                            }
                            if iid == ::windows_core::DYNAMIC_CAST_IID {
                                (interface as *mut *const dyn core::any::Any)
                                    .write(
                                        self as &dyn ::core::any::Any as *const dyn ::core::any::Any,
                                    );
                                return ::windows_core::HRESULT(0);
                            }
                            let tear_off_ptr = self
                                .count
                                .query(&iid, &self.identity as *const _ as *mut _);
                            if !tear_off_ptr.is_null() {
                                *interface = tear_off_ptr;
                                return ::windows_core::HRESULT(0);
                            }
                            *interface = ::core::ptr::null_mut();
                            return ::windows_core::imp::E_NOINTERFACE;
                        };
                        if true {
                            if !!interface_ptr.is_null() {
                                ::core::panicking::panic(
                                    "assertion failed: !interface_ptr.is_null()",
                                )
                            }
                        }
                        *interface = interface_ptr as *mut ::core::ffi::c_void;
                        self.count.add_ref();
                        return ::windows_core::HRESULT(0);
                    }
                }
            }
            impl ::windows_core::ComObjectInner for ToastActivationManager {
                type Outer = ToastActivationManager_Impl;
                fn into_object(self) -> ::windows_core::ComObject<Self> {
                    let boxed = ::windows_core::imp::Box::<
                        ToastActivationManager_Impl,
                    >::new(self.into_outer());
                    unsafe {
                        let ptr = ::windows_core::imp::Box::into_raw(boxed);
                        ::windows_core::ComObject::from_raw(
                            ::core::ptr::NonNull::new_unchecked(ptr),
                        )
                    }
                }
            }
            impl ::core::convert::From<ToastActivationManager>
            for ::windows_core::IUnknown {
                #[inline(always)]
                fn from(this: ToastActivationManager) -> Self {
                    let com_object = ::windows_core::ComObject::new(this);
                    com_object.into_interface()
                }
            }
            impl ::core::convert::From<ToastActivationManager>
            for ::windows_core::IInspectable {
                #[inline(always)]
                fn from(this: ToastActivationManager) -> Self {
                    let com_object = ::windows_core::ComObject::new(this);
                    com_object.into_interface()
                }
            }
            impl ::core::convert::From<ToastActivationManager>
            for INotificationActivationCallback {
                #[inline(always)]
                fn from(this: ToastActivationManager) -> Self {
                    let com_object = ::windows_core::ComObject::new(this);
                    com_object.into_interface()
                }
            }
            impl ::windows_core::ComObjectInterface<::windows_core::IUnknown>
            for ToastActivationManager_Impl {
                #[inline(always)]
                fn as_interface_ref(
                    &self,
                ) -> ::windows_core::InterfaceRef<'_, ::windows_core::IUnknown> {
                    unsafe {
                        let interface_ptr = &self.identity;
                        ::core::mem::transmute(interface_ptr)
                    }
                }
            }
            impl ::windows_core::ComObjectInterface<::windows_core::IInspectable>
            for ToastActivationManager_Impl {
                #[inline(always)]
                fn as_interface_ref(
                    &self,
                ) -> ::windows_core::InterfaceRef<'_, ::windows_core::IInspectable> {
                    unsafe {
                        let interface_ptr = &self.identity;
                        ::core::mem::transmute(interface_ptr)
                    }
                }
            }
            #[allow(clippy::needless_lifetimes)]
            impl ::windows_core::ComObjectInterface<INotificationActivationCallback>
            for ToastActivationManager_Impl {
                #[inline(always)]
                fn as_interface_ref(
                    &self,
                ) -> ::windows_core::InterfaceRef<'_, INotificationActivationCallback> {
                    unsafe {
                        ::core::mem::transmute(&self.interface1_inotificationactivat)
                    }
                }
            }
            impl ::windows_core::AsImpl<ToastActivationManager>
            for INotificationActivationCallback {
                #[inline(always)]
                unsafe fn as_impl_ptr(
                    &self,
                ) -> ::core::ptr::NonNull<ToastActivationManager> {
                    unsafe {
                        let this = ::windows_core::Interface::as_raw(self);
                        let this = (this as *mut *mut ::core::ffi::c_void)
                            .sub(1 + 0usize) as *mut ToastActivationManager_Impl;
                        ::core::ptr::NonNull::new_unchecked(
                            &raw const (*this).this as *const ToastActivationManager
                                as *mut ToastActivationManager,
                        )
                    }
                }
            }
            impl INotificationActivationCallback_Impl for ToastActivationManager_Impl {
                fn Activate(
                    &self,
                    _appusermodelid: &windows_core::PCWSTR,
                    invokedargs: &windows_core::PCWSTR,
                    _data: *const NOTIFICATION_USER_INPUT_DATA,
                    _count: u32,
                ) -> windows_core::Result<()> {
                    {
                        ::std::io::_print(format_args!("Called\n"));
                    };
                    if invokedargs.is_null() {
                        {
                            ::std::io::_print(
                                format_args!(
                                    "Toast activated (default click), no arguments.\n",
                                ),
                            );
                        };
                    } else {
                        let args_str = unsafe { invokedargs.to_string() };
                        {
                            ::std::io::_print(
                                format_args!(
                                    "Toast activated with arguments: {0}\n",
                                    args_str.unwrap_or_default(),
                                ),
                            );
                        };
                    }
                    Ok(())
                }
            }
        }
        pub struct ToastsNotifier {
            _inner: ToastNotifier,
            app_id: Arc<Box<str>>,
        }
        impl ToastsNotifier {
            pub fn new<T: Into<String>>(app_id: T) -> Result<Self, NotifError> {
                Self::new_inner(app_id, None)
            }
            pub(crate) fn new_inner<T: Into<String>>(
                app_id: T,
                guid: Option<u128>,
            ) -> Result<Self, NotifError> {
                let app_id = app_id.into();
                if let Some(guid) = guid {
                    let app_id = app_id.clone();
                    thread::spawn(move || {
                        unsafe {
                            SetCurrentProcessExplicitAppUserModelID(
                                    &HSTRING::from(app_id.as_str()),
                                )
                                .unwrap();
                            _ = CoInitializeEx(None, COINIT_APARTMENTTHREADED)
                                .ok()
                                .unwrap();
                            let factory: IUnknown = ToastActivationManager.into();
                            CoRegisterClassObject(
                                    &GUID::from_u128(guid),
                                    &factory,
                                    CLSCTX_LOCAL_SERVER,
                                    REGCLS_MULTIPLEUSE,
                                )
                                .unwrap();
                            let mut msg = MSG::default();
                            while GetMessageW(&mut msg, None, 0, 0).into() {
                                {
                                    ::std::io::_print(format_args!("Got Msg\n"));
                                };
                                _ = TranslateMessage(&msg);
                                DispatchMessageW(&msg);
                            }
                        };
                    });
                }
                let string: String = app_id.clone();
                let string = string.into_boxed_str();
                let id = HSTRING::from(string.as_ref());
                let _inner = ToastNotificationManager::CreateToastNotifierWithId(&id)?;
                Ok(Self {
                    _inner,
                    app_id: Arc::new(string),
                })
            }
            pub fn manager(&self) -> Result<ToastsManager, NotifError> {
                Ok(ToastsManager {
                    inner: (ToastNotificationManager::History()?),
                    app_id: self.app_id.clone(),
                })
            }
            pub fn update(
                &self,
                data: &NotificationDataSet,
                group: &str,
                tag: &str,
            ) -> Result<NotificationUpdateResult, NotifError> {
                let raw: &NotificationData = data.inner_win32_type();
                Ok(self._inner.UpdateWithTagAndGroup(raw, &tag.into(), &group.into())?)
            }
            pub(crate) fn get_raw_handle(&self) -> &ToastNotifier {
                &self._inner
            }
            pub unsafe fn as_raw(&self) -> &ToastNotifier {
                &self._inner
            }
        }
        pub struct ToastsManager {
            pub(crate) inner: ToastNotificationHistory,
            pub app_id: Arc<Box<str>>,
        }
        #[automatically_derived]
        impl ::core::fmt::Debug for ToastsManager {
            #[inline]
            fn fmt(&self, f: &mut ::core::fmt::Formatter) -> ::core::fmt::Result {
                ::core::fmt::Formatter::debug_struct_field2_finish(
                    f,
                    "ToastsManager",
                    "inner",
                    &self.inner,
                    "app_id",
                    &&self.app_id,
                )
            }
        }
        #[automatically_derived]
        impl ::core::clone::Clone for ToastsManager {
            #[inline]
            fn clone(&self) -> ToastsManager {
                ToastsManager {
                    inner: ::core::clone::Clone::clone(&self.inner),
                    app_id: ::core::clone::Clone::clone(&self.app_id),
                }
            }
        }
        impl ToastsManager {
            pub unsafe fn as_raw(&self) -> &ToastNotificationHistory {
                &self.inner
            }
            /// Clear all notifications from this application
            pub fn clear(&self) -> Result<(), NotifError> {
                Ok(self.inner.Clear()?)
            }
            /// Clears all notifications sent by another app
            /// from the same app package
            ///
            /// ## WARNING
            /// This is probably not meant for Win32 Apps but we're not sure
            pub fn clear_appid(&self, app_id: &str) -> Result<(), NotifError> {
                let hstr = HSTRING::from(app_id);
                Ok(self.inner.ClearWithId(&hstr)?)
            }
            /// Removes a notification identified by tag, group, notif_id
            pub fn remove_notification(
                &self,
                tag: &str,
                group: &str,
                notif_id: &str,
            ) -> Result<(), NotifError> {
                let hstr = HSTRING::from(tag);
                let group = HSTRING::from(group);
                let id = HSTRING::from(notif_id);
                Ok(self.inner.RemoveGroupedTagWithId(&hstr, &group, &id)?)
            }
            /// Removes a notification identified by tag, group
            pub fn remove_notification_with_gt(
                &self,
                tag: &str,
                group: &str,
            ) -> Result<(), NotifError> {
                let hstr = HSTRING::from(tag);
                let group = HSTRING::from(group);
                Ok(self.inner.RemoveGroupedTag(&hstr, &group)?)
            }
            /// Removes a notification identified by tag only
            pub fn remove_notification_with_tag(
                &self,
                tag: &str,
            ) -> Result<(), NotifError> {
                let hstr = HSTRING::from(tag);
                Ok(self.inner.Remove(&hstr)?)
            }
            /// Removes a group of notifications identified by the group id
            pub fn remove_group(&self, group: &str) -> Result<(), NotifError> {
                let hstr = HSTRING::from(group);
                Ok(self.inner.RemoveGroup(&hstr)?)
            }
            /// Removes a group of notifications identified by the group id for **another app**
            /// from the same app package
            ///
            /// ## WARNING
            /// This is probably not meant for Win32 Apps but we're not sure
            pub fn remove_group_from_appid(
                &self,
                group: &str,
                app_id: &str,
            ) -> Result<(), NotifError> {
                let app_id = HSTRING::from(app_id);
                let hstr = HSTRING::from(group);
                Ok(self.inner.RemoveGroupWithId(&hstr, &app_id)?)
            }
            /// Gets notification history as PartialNotification objects
            pub fn get_notification_history(
                &self,
            ) -> Result<Vec<OwnedPartialNotification>, NotifError> {
                let data = self.inner.GetHistory()?;
                let da = data
                    .into_iter()
                    .map(|x| OwnedPartialNotification {
                        notif: x,
                    })
                    .collect::<Vec<_>>();
                Ok(da)
            }
            /// Gets notification history as PartialNotification objects for **another app**
            /// from the same app package
            ///
            /// ## WARNING
            /// This is probably not meant for Win32 Apps but we're not sure
            pub fn get_notification_history_with_id(
                &self,
                app_id: &str,
            ) -> Result<Vec<OwnedPartialNotification>, NotifError> {
                let appid = HSTRING::from(app_id);
                let data = self.inner.GetHistoryWithId(&appid)?;
                let da = data
                    .into_iter()
                    .map(|x| OwnedPartialNotification {
                        notif: x,
                    })
                    .collect::<Vec<_>>();
                Ok(da)
            }
        }
    }
    use std::time::Duration;
    pub use data::NotificationDataSet;
    pub use handler::{
        NotificationActivatedEventHandler, NotificationDismissedEventHandler,
        NotificationFailedEventHandler,
    };
    pub use notification::{Notification, NotificationBuilder};
    pub use notifier::ToastsNotifier;
    use windows::{
        core::HSTRING, Foundation::{DateTime, IReference, PropertyValue},
        Globalization::Calendar,
        UI::Notifications::{
            NotificationMirroring as ToastNotificationMirroring, ToastNotification,
            ToastNotificationPriority,
        },
    };
    use windows_core::Interface;
    use crate::NotifError;
    pub enum NotificationPriority {
        Default,
        High,
    }
    pub enum NotificationMirroring {
        Allowed,
        Disallowed,
    }
    pub(crate) trait ToXML {
        fn to_xml(&self) -> String;
    }
    pub trait NotificationImpl {
        fn notif(&self) -> &ToastNotification;
    }
    pub trait ManageNotification {
        fn get_xml_content(&self) -> Option<String>;
        fn priority(&self) -> Result<NotificationPriority, NotifError>;
        fn set_priority(&self, priority: NotificationPriority) -> Result<(), NotifError>;
        fn notification_mirroring(&self) -> Result<NotificationMirroring, NotifError>;
        fn set_notification_mirroring(
            &self,
            mirroring: NotificationMirroring,
        ) -> Result<(), NotifError>;
        fn set_expiration(&self, expires: Duration) -> Result<(), NotifError>;
        fn set_activated_handler(
            &self,
            handler: NotificationActivatedEventHandler,
        ) -> Result<i64, NotifError>;
        fn remove_activated_handler(&self, token: i64) -> Result<(), NotifError>;
        fn set_dismissed_handler(
            &self,
            handler: NotificationDismissedEventHandler,
        ) -> Result<i64, NotifError>;
        fn remove_dismissed_handler(&self, token: i64) -> Result<(), NotifError>;
        fn set_failed_handler(
            &self,
            handler: NotificationFailedEventHandler,
        ) -> Result<i64, NotifError>;
        fn remove_failed_handler(&self, token: i64) -> Result<(), NotifError>;
        fn get_tag(&self) -> Result<String, NotifError>;
        fn set_tag(&self, tag: String) -> Result<(), NotifError>;
        fn get_group(&self) -> Result<String, NotifError>;
        fn set_group(&self, group: String) -> Result<(), NotifError>;
        fn get_remote_id(&self) -> Result<String, NotifError>;
        fn set_remote_id(&self, remote: String) -> Result<(), NotifError>;
        fn suppress_popup(&self) -> Result<bool, NotifError>;
        fn set_suppress_popup(&self, value: bool) -> Result<(), NotifError>;
        fn expires_on_reboot(&self) -> Result<bool, NotifError>;
        fn set_expires_on_reboot(&self, value: bool) -> Result<(), NotifError>;
    }
    impl<T: NotificationImpl> ManageNotification for T {
        fn get_xml_content(&self) -> Option<String> {
            Some(self.notif().Content().ok()?.GetXml().ok()?.to_string())
        }
        fn set_expiration(&self, expires: Duration) -> Result<(), NotifError> {
            let calendar = Calendar::new()?;
            if expires.as_secs() > i32::MAX as u64 {
                return Err(NotifError::DurationTooLong);
            }
            calendar.AddSeconds(expires.as_secs() as i32)?;
            let dt = calendar.GetDateTime()?;
            self.notif()
                .SetExpirationTime(
                    &PropertyValue::CreateDateTime(dt)?.cast::<IReference<DateTime>>()?,
                )?;
            Ok(())
        }
        fn priority(&self) -> Result<NotificationPriority, NotifError> {
            let priority = self.notif().Priority()?.0;
            let def = ToastNotificationPriority::Default.0;
            let high = ToastNotificationPriority::High.0;
            if priority == def {
                return Ok(NotificationPriority::Default);
            } else if priority == high {
                return Ok(NotificationPriority::High);
            }
            Err(NotifError::UnknownAndImpossible)
        }
        fn set_priority(
            &self,
            priority: NotificationPriority,
        ) -> Result<(), NotifError> {
            let priority = match priority {
                NotificationPriority::Default => ToastNotificationPriority::Default,
                NotificationPriority::High => ToastNotificationPriority::High,
            };
            Ok(self.notif().SetPriority(priority)?)
        }
        fn notification_mirroring(&self) -> Result<NotificationMirroring, NotifError> {
            let mirroring = self.notif().NotificationMirroring()?.0;
            if mirroring == ToastNotificationMirroring::Allowed.0 {
                return Ok(NotificationMirroring::Allowed);
            } else if mirroring == ToastNotificationMirroring::Disabled.0 {
                return Ok(NotificationMirroring::Disallowed);
            }
            Err(NotifError::UnknownAndImpossible)
        }
        fn set_notification_mirroring(
            &self,
            mirroring: NotificationMirroring,
        ) -> Result<(), NotifError> {
            let mirroring = match mirroring {
                NotificationMirroring::Allowed => ToastNotificationMirroring::Allowed,
                NotificationMirroring::Disallowed => ToastNotificationMirroring::Disabled,
            };
            Ok(self.notif().SetNotificationMirroring(mirroring)?)
        }
        fn set_activated_handler(
            &self,
            handler: NotificationActivatedEventHandler,
        ) -> Result<i64, NotifError> {
            Ok(self.notif().Activated(&handler.handler)?)
        }
        fn remove_activated_handler(&self, token: i64) -> Result<(), NotifError> {
            Ok(self.notif().RemoveActivated(token)?)
        }
        fn set_dismissed_handler(
            &self,
            handler: NotificationDismissedEventHandler,
        ) -> Result<i64, NotifError> {
            Ok(self.notif().Dismissed(&handler.handler)?)
        }
        fn remove_dismissed_handler(&self, token: i64) -> Result<(), NotifError> {
            Ok(self.notif().RemoveDismissed(token)?)
        }
        fn set_failed_handler(
            &self,
            handler: NotificationFailedEventHandler,
        ) -> Result<i64, NotifError> {
            Ok(self.notif().Failed(&handler.handler)?)
        }
        fn remove_failed_handler(&self, token: i64) -> Result<(), NotifError> {
            Ok(self.notif().RemoveFailed(token)?)
        }
        fn get_tag(&self) -> Result<String, NotifError> {
            Ok(self.notif().Tag()?.to_string())
        }
        fn set_tag(&self, data: String) -> Result<(), NotifError> {
            Ok(self.notif().SetTag(&HSTRING::from(data))?)
        }
        fn get_group(&self) -> Result<String, NotifError> {
            Ok(self.notif().Group()?.to_string())
        }
        fn set_group(&self, data: String) -> Result<(), NotifError> {
            Ok(self.notif().SetGroup(&HSTRING::from(data))?)
        }
        fn get_remote_id(&self) -> Result<String, NotifError> {
            Ok(self.notif().RemoteId()?.to_string())
        }
        fn set_remote_id(&self, data: String) -> Result<(), NotifError> {
            Ok(self.notif().SetRemoteId(&HSTRING::from(data))?)
        }
        fn expires_on_reboot(&self) -> Result<bool, NotifError> {
            Ok(self.notif().ExpiresOnReboot()?)
        }
        fn set_expires_on_reboot(&self, data: bool) -> Result<(), NotifError> {
            Ok(self.notif().SetExpiresOnReboot(data)?)
        }
        fn suppress_popup(&self) -> Result<bool, NotifError> {
            Ok(self.notif().SuppressPopup()?)
        }
        fn set_suppress_popup(&self, data: bool) -> Result<(), NotifError> {
            Ok(self.notif().SetSuppressPopup(data)?)
        }
    }
}
use std::{error::Error, fmt::Display};
pub use structs::*;
pub enum NotifError {
    WindowsCore(windows::core::Error),
    DurationTooLong,
    UnknownAndImpossible,
}
#[automatically_derived]
impl ::core::fmt::Debug for NotifError {
    #[inline]
    fn fmt(&self, f: &mut ::core::fmt::Formatter) -> ::core::fmt::Result {
        match self {
            NotifError::WindowsCore(__self_0) => {
                ::core::fmt::Formatter::debug_tuple_field1_finish(
                    f,
                    "WindowsCore",
                    &__self_0,
                )
            }
            NotifError::DurationTooLong => {
                ::core::fmt::Formatter::write_str(f, "DurationTooLong")
            }
            NotifError::UnknownAndImpossible => {
                ::core::fmt::Formatter::write_str(f, "UnknownAndImpossible")
            }
        }
    }
}
impl Display for NotifError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_fmt(format_args!("{0:?}", self))
    }
}
impl Error for NotifError {}
impl From<windows::core::Error> for NotifError {
    fn from(value: windows::core::Error) -> Self {
        Self::WindowsCore(value)
    }
}
