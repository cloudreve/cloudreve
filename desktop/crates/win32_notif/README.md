# Win32 Notif

[![Crates.io Version](https://img.shields.io/crates/v/win32_notif?logo=Docs.rs)](https://docs.rs/win32_notif)

A lightweight crate to help you to compose beautiful notifications for Windows OS.

This crate aims for **100%** coverage of the WinRT Toast api as much as possible.

Thankfully we are quite near that goal due to our unique approach to notification content: **widgets**

You declare your own style, however you like as long as the XML Supports it.

## Basic Usage

```rust
use std::{path::absolute, thread::sleep, time::Duration};

use win32_notif::{
  NotificationBuilder, NotificationDataSet, notification::visual::{Image, Placement, Text, image::{AdaptiveImageAlign, ImageCrop}, text::HintStyle}, notifier::ToastsNotifier
};

fn main() {
  let notifier = ToastsNotifier::new("Microsoft.Windows.Explorer").unwrap();

  let notif = NotificationBuilder::new()
    .visual(
      Text::create(0, "Welcome to \"win32_notif\"!! 👋")
        .align_center(true)
        .wrap(true)
        .with_style(HintStyle::Title)
    )
    .visual(
      Text::create_binded(1, "desc")
        .align_center(true)
        .wrap(true)
        .with_style(HintStyle::Body)
    )
    .value("desc", "Data binding works as well {WOW}!")
    .build(0, &notifier, "01", "readme")
    .unwrap();

  notif.show()
    .unwrap();

  sleep(Duration::from_secs(1));

  let data = NotificationDataSet::new().unwrap();

  data.insert("desc", "Hello, the message is edited").unwrap();

  notifier.update(&data, "readme", "01").unwrap();
}
```

## What is implemented

We've actually implemented a lot of the Notification APIs

### Containers

- Text
- Image
- Progressbar
- Groups
- Subgroups

### Handlers

- Foreground OnActivated
- Foreground OnError
- Foregrounf OnDismissed

### Utility

- Notification Updating
- Data Binding (so that you can update notification content)
- Notification Duration
- Scenarios
- Command
- Actions
- Inputs
- Selections
- Visual
- **_Idiomatic Rust Builder Style (with\_... methods)_**

**_and a lot of other things... 🎉_**

## Future Project Plan

- COM Activator
- Background Activation Handling

...and that's it
