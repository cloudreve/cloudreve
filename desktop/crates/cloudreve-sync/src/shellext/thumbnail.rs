use crate::drive::commands::ManagerCommand;
use crate::drive::manager::DriveManager;
use bytes::Bytes;
use std::path::PathBuf;
use std::sync::{Arc, Mutex};
use windows::{
    Graphics::Imaging::{BitmapAlphaMode, BitmapDecoder, BitmapPixelFormat, BitmapTransform},
    Storage::Streams::{DataWriter, InMemoryRandomAccessStream},
    Win32::{Foundation::*, Graphics::Gdi, System::Com::*, UI::Shell::*},
    core::*,
};

pub const CLSID_THUMBNAIL_PROVIDER: GUID = GUID::from_u128(0x3d781652_78c5_4038_87a4_ec5940ab560a);

#[implement(IThumbnailProvider, IInitializeWithItem)]
pub struct ThumbnailProvider {
    drive_manager: Arc<DriveManager>,
    path: Arc<Mutex<Option<PathBuf>>>,
}

impl ThumbnailProvider {
    pub fn new(drive_manager: Arc<DriveManager>) -> Self {
        Self {
            drive_manager,
            path: Arc::new(Mutex::new(None)),
        }
    }

    /// Convert image bytes to HBITMAP
    /// Supports JPG, PNG, WebP and other formats supported by Windows Imaging Component
    fn bytes_to_hbitmap(
        &self,
        image_bytes: &Bytes,
        max_size: u32,
    ) -> Result<(Gdi::HBITMAP, WTS_ALPHATYPE)> {
        unsafe {
            // Create an in-memory random access stream
            let stream = InMemoryRandomAccessStream::new()?;

            // Write the image bytes to the stream
            let writer = DataWriter::CreateDataWriter(&stream)?;
            writer.WriteBytes(image_bytes.as_ref())?;
            writer.StoreAsync()?.get()?;
            writer.FlushAsync()?.get()?;

            // Seek to the beginning of the stream
            stream.Seek(0)?;

            // Create a bitmap decoder
            let decoder = BitmapDecoder::CreateAsync(&stream)?.get()?;

            // Get the frame
            let frame = decoder.GetFrameAsync(0)?.get()?;

            // Get dimensions
            let width = frame.PixelWidth()?;
            let height = frame.PixelHeight()?;

            // Calculate target dimensions
            let (target_width, target_height) = if width > max_size || height > max_size {
                let aspect_ratio = width as f32 / height as f32;
                if width > height {
                    (max_size, (max_size as f32 / aspect_ratio) as u32)
                } else {
                    ((max_size as f32 * aspect_ratio) as u32, max_size)
                }
            } else {
                (width, height)
            };

            // Create transform for scaling if needed
            let transform = BitmapTransform::new()?;
            if target_width != width || target_height != height {
                transform.SetScaledWidth(target_width)?;
                transform.SetScaledHeight(target_height)?;
            }

            // Get pixel data in BGRA8 format with premultiplied alpha
            let pixel_provider = frame
                .GetPixelDataTransformedAsync(
                    BitmapPixelFormat::Bgra8,
                    BitmapAlphaMode::Premultiplied,
                    Some(&transform),
                    windows::Graphics::Imaging::ExifOrientationMode::RespectExifOrientation,
                    windows::Graphics::Imaging::ColorManagementMode::DoNotColorManage,
                )?
                .get()?;

            let pixel_data = pixel_provider.DetachPixelData()?;

            // Create BITMAPINFO structure
            let bmi = Gdi::BITMAPINFO {
                bmiHeader: Gdi::BITMAPINFOHEADER {
                    biSize: std::mem::size_of::<Gdi::BITMAPINFOHEADER>() as u32,
                    biWidth: target_width as i32,
                    biHeight: -(target_height as i32), // Negative for top-down DIB
                    biPlanes: 1,
                    biBitCount: 32,
                    biCompression: Gdi::BI_RGB.0 as u32,
                    biSizeImage: 0,
                    biXPelsPerMeter: 0,
                    biYPelsPerMeter: 0,
                    biClrUsed: 0,
                    biClrImportant: 0,
                },
                bmiColors: [Gdi::RGBQUAD::default()],
            };

            // Create DIB section
            let mut bits: *mut std::ffi::c_void = std::ptr::null_mut();
            let hdc = Gdi::GetDC(None);
            let hbitmap =
                Gdi::CreateDIBSection(hdc, &bmi, Gdi::DIB_RGB_COLORS, &mut bits, None, 0)?;

            if !bits.is_null() && !pixel_data.is_empty() {
                // Copy pixel data to the bitmap
                let dest_slice = std::slice::from_raw_parts_mut(bits as *mut u8, pixel_data.len());
                dest_slice.copy_from_slice(&pixel_data);
            }

            Gdi::ReleaseDC(None, hdc);

            // Return premultiplied alpha type
            let alpha_type = WTSAT_ARGB;

            Ok((hbitmap, alpha_type))
        }
    }
}

impl IThumbnailProvider_Impl for ThumbnailProvider_Impl {
    fn GetThumbnail(
        &self,
        cx: u32,
        phbmp: *mut Gdi::HBITMAP,
        pdwalpha: *mut WTS_ALPHATYPE,
    ) -> Result<()> {
        let path = self
            .path
            .lock()
            .map_err(|_| Error::from(E_FAIL))?
            .clone()
            .ok_or_else(|| Error::from(E_FAIL))?;

        tracing::trace!(target: "shellext::thumbnail", path = ?path, size = cx, "GetThumbnail called");

        let command_tx = self.drive_manager.get_command_sender();
        let (response_tx, response_rx) = tokio::sync::oneshot::channel();
        if let Err(e) = command_tx.send(ManagerCommand::GenerateThumbnail {
            path: path.clone(),
            response: response_tx,
        }) {
            tracing::error!(target: "shellext::thumbnail", error = %e, "Failed to send GenerateThumbnail command");
            return Err(Error::from(E_FAIL));
        }

        let image_bytes = response_rx
            .blocking_recv()
            .map_err(|e| {
                tracing::error!(target: "shellext::thumbnail", error = %e, "Failed to receive GenerateThumbnail response");
                Error::from(E_FAIL)
            })?
            .map_err(|e| {
                tracing::error!(target: "shellext::thumbnail", error = %e, "GenerateThumbnail command failed");
                Error::from(E_FAIL)
            })?;

        tracing::trace!(target: "shellext::thumbnail", bytes_len = image_bytes.len(), "Received image bytes");

        // Convert image bytes to HBITMAP
        let (hbitmap, alpha_type) = self.bytes_to_hbitmap(&image_bytes, cx).map_err(|e| {
            tracing::error!(target: "shellext::thumbnail", error = ?e, "Failed to convert image bytes to HBITMAP");
            e
        })?;

        unsafe {
            // Return the bitmap handle and alpha type
            if !phbmp.is_null() {
                *phbmp = hbitmap;
            }
            if !pdwalpha.is_null() {
                *pdwalpha = alpha_type;
            }
        }

        tracing::trace!(target: "shellext::thumbnail", "Thumbnail generated successfully");

        Ok(())
    }
}

impl IInitializeWithItem_Impl for ThumbnailProvider_Impl {
    fn Initialize(&self, _psi: Option<&IShellItem>, _riid: u32) -> Result<()> {
        tracing::trace!(target: "shellext::thumbnail", riid = _riid, "Initializing thumbnail provider");
        let Some(psi) = _psi else {
            return Err(Error::from(E_INVALIDARG));
        };
        unsafe {
            let path = psi.GetDisplayName(SIGDN_FILESYSPATH)?.to_string()?;
            let path = PathBuf::from(path);
            self.path
                .lock()
                .map_err(|_| Error::from(E_FAIL))?
                .replace(path);
        }
        Ok(())
    }
}

// Class factory for creating instances of our context menu handler
#[implement(IClassFactory)]
pub struct ThumbnailProviderFactory {
    drive_manager: Arc<DriveManager>,
}

impl ThumbnailProviderFactory {
    pub fn new(drive_manager: Arc<DriveManager>) -> Self {
        Self { drive_manager }
    }
}

impl IClassFactory_Impl for ThumbnailProviderFactory_Impl {
    fn CreateInstance(
        &self,
        outer: Option<&IUnknown>,
        iid: *const GUID,
        result: *mut *mut core::ffi::c_void,
    ) -> Result<()> {
        if outer.is_some() {
            return Err(Error::from(CLASS_E_NOAGGREGATION));
        }

        let handler = ThumbnailProvider::new(self.drive_manager.clone());
        let handler: IUnknown = handler.into();

        unsafe { handler.query(iid, result).ok() }
    }

    fn LockServer(&self, _lock: BOOL) -> Result<()> {
        Ok(())
    }
}
