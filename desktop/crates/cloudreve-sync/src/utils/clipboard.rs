//! Minimal clipboard helpers. Only Windows sets real clipboard text; other
//! platforms no-op so callers can stay unconditional.

#[cfg(windows)]
pub fn set_text(text: &str) -> anyhow::Result<()> {
    use windows::Win32::{
        Foundation::HANDLE,
        System::{
            DataExchange::{CloseClipboard, EmptyClipboard, OpenClipboard, SetClipboardData},
            Memory::{GMEM_MOVEABLE, GlobalAlloc, GlobalLock, GlobalUnlock},
            Ole::CF_UNICODETEXT,
        },
    };

    unsafe {
        OpenClipboard(None)?;
        let result = (|| -> anyhow::Result<()> {
            EmptyClipboard()?;
            let wide: Vec<u16> = text.encode_utf16().chain(std::iter::once(0)).collect();
            let hglobal = GlobalAlloc(GMEM_MOVEABLE, wide.len() * 2)?;
            let dst = GlobalLock(hglobal);
            if dst.is_null() {
                return Err(anyhow::anyhow!("GlobalLock failed"));
            }
            std::ptr::copy_nonoverlapping(wide.as_ptr(), dst as *mut u16, wide.len());
            let _ = GlobalUnlock(hglobal);
            SetClipboardData(CF_UNICODETEXT.0 as u32, HANDLE(hglobal.0))?;
            Ok(())
        })();
        let _ = CloseClipboard();
        result
    }
}

#[cfg(not(windows))]
pub fn set_text(_text: &str) -> anyhow::Result<()> {
    tracing::warn!(target: "utils::clipboard", "Clipboard set_text is not supported on this platform");
    Ok(())
}
