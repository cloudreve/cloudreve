use super::context_menu::*;
use crate::drive::manager::DriveManager;
use crate::shellext::custom_state::{CLSID_CUSTOM_STATE_HANDLER, CustomStateHandlerFactory};
use crate::shellext::status_ui::{
    CLSID_STATUS_UI_HANDLER, StatusUIHandlerFactoryFactory,
};
use crate::shellext::thumbnail::{CLSID_THUMBNAIL_PROVIDER, ThumbnailProviderFactory};
use crate::shellext::toast::{CLSID_TOAST_ACTIVATOR, ToastActivatorFactory};
use std::sync::{Arc, mpsc};
use std::thread;
use windows::Win32::System::Com::{COINIT_MULTITHREADED, CoWaitForMultipleHandles};
use windows::Win32::System::Threading::CreateEventW;
use windows::{
    Win32::{Foundation::*, System::Com::*},
    core::*,
};

pub fn init_and_start_service_task(drive_manager: Arc<DriveManager>) -> ServiceHandle {
    let (tx, rx) = mpsc::channel();

    let mut services = ShellServices::new(drive_manager);
    let handle: thread::JoinHandle<()> = thread::spawn(move || {
        // Step 1: Initialize COM
        if let Err(e) = services.init_com() {
            tracing::error!(target: "shellext::shell_service", "Failed to initialize COM: {:?}", e);
            let _ = tx.send(Err(e));
            return;
        }

        // Step 2: Initialize handlers
        if let Err(e) = services.init_and_start_custom_state_handler() {
            tracing::error!(target: "shellext::shell_service", "Failed to initialize custom state handler: {:?}", e);
            let _ = tx.send(Err(e));
            return;
        }

        if let Err(e) = services.init_and_start_view_online_handler() {
            tracing::error!(target: "shellext::shell_service", "Failed to initialize view online handler: {:?}", e);
            let _ = tx.send(Err(e));
            return;
        }

        if let Err(e) = services.init_and_start_status_ui_handler() {
            tracing::error!(target: "shellext::shell_service", "Failed to initialize status ui handler: {:?}", e);
            let _ = tx.send(Err(e));
            return;
        }

        if let Err(e) = services.init_and_start_thumbnail_provider_handler() {
            tracing::error!(target: "shellext::shell_service", "Failed to initialize thumbnail provider handler: {:?}", e);
            let _ = tx.send(Err(e));
            return;
        }

        if let Err(e) = services.init_and_start_toast_handler() {
            tracing::error!(target: "shellext::shell_service", "Failed to initialize toast handler: {:?}", e);
            let _ = tx.send(Err(e));
            return;
        }

        // Notify that initialization is complete
        let _ = tx.send(Ok(()));

        // Step 3: Start message loop
        if let Err(e) = services.run_message_loop() {
            tracing::error!(target: "shellext::shell_service", "Error in message loop: {:?}", e);
        }
    });

    ServiceHandle {
        thread: Some(handle),
        init_result: rx,
    }
}

pub struct ServiceHandle {
     #[allow(dead_code)]
    thread: Option<thread::JoinHandle<()>>,
    init_result: mpsc::Receiver<windows::core::Result<()>>,
}

impl ServiceHandle {
    pub fn wait_for_init(&mut self) -> windows::core::Result<()> {
        self.init_result
            .recv()
            .map_err(|_| windows::core::Error::from(windows::Win32::Foundation::E_FAIL))?
    }
}

impl Drop for ServiceHandle {
    fn drop(&mut self) {
        tracing::info!(target: "shellext::shell_service", "Shutting down shell services thread...");
    }
}

// Shell services - registers COM objects for Windows Shell integration
pub struct ShellServices {
    cookies: Vec<u32>,
    drive_manager: Arc<DriveManager>,
}

impl ShellServices {
    pub fn new(drive_manager: Arc<DriveManager>) -> Self {
        Self {
            cookies: Vec::new(),
            drive_manager,
        }
    }

    pub fn init_com(&mut self) -> Result<()> {
        unsafe {
            CoInitializeEx(None, COINIT_MULTITHREADED).ok()?;
        }
        Ok(())
    }

    pub fn init_and_start_thumbnail_provider_handler(&mut self) -> Result<()> {
        tracing::info!(target: "shellext::thumbnail", "Initializing Shell Services (Thumbnail Provider Handler)...");

        unsafe {
            let factory: IClassFactory =
                ThumbnailProviderFactory::new(self.drive_manager.clone()).into();
            let cookie = CoRegisterClassObject(
                &CLSID_THUMBNAIL_PROVIDER,
                &factory,
                CLSCTX_LOCAL_SERVER,
                REGCLS_MULTIPLEUSE,
            )?;

            self.cookies.push(cookie);
            tracing::info!(target: "shellext::thumbnail", "Thumbnail Provider Handler registered with cookie: {}", cookie);
        }

        Ok(())
    }

    pub fn init_and_start_toast_handler(&mut self) -> Result<()> {
        tracing::info!(target: "shellext::toast", "Initializing Shell Services (Toast Handler)...");

        unsafe {
            // Create and register the class factory
            let factory: IClassFactory =
                ToastActivatorFactory::new(self.drive_manager.clone()).into();

            let cookie = CoRegisterClassObject(
                &CLSID_TOAST_ACTIVATOR,
                &factory,
                CLSCTX_LOCAL_SERVER,
                REGCLS_MULTIPLEUSE,
            )?;

            self.cookies.push(cookie);
            tracing::info!(target: "shellext::toast", "Toast Handler registered with cookie: {}", cookie);
        }

        Ok(())
    }

    pub fn init_and_start_status_ui_handler(&mut self) -> Result<()> {
        tracing::info!(target: "shellext::status_ui", "Initializing Shell Services (Status UI Handler)...");

        unsafe {
            let factory: IClassFactory =
                StatusUIHandlerFactoryFactory::new(self.drive_manager.clone()).into();

            let cookie = CoRegisterClassObject(
                &CLSID_STATUS_UI_HANDLER,
                &factory,
                CLSCTX_LOCAL_SERVER,
                REGCLS_MULTIPLEUSE,
            )?;

            self.cookies.push(cookie);
            tracing::info!(target: "shellext::status_ui", "Status UI Handler registered with cookie: {}", cookie);
        }

        Ok(())
    }

    pub fn init_and_start_custom_state_handler(&mut self) -> Result<()> {
        tracing::info!(target: "shellext::custom_state", "Initializing Shell Services (Custom State Handler)...");

        unsafe {
            let factory: IClassFactory =
                CustomStateHandlerFactory::new(self.drive_manager.clone()).into();

            let cookie = CoRegisterClassObject(
                &CLSID_CUSTOM_STATE_HANDLER,
                &factory,
                CLSCTX_LOCAL_SERVER,
                REGCLS_MULTIPLEUSE,
            )?;

            self.cookies.push(cookie);
            tracing::info!(target: "shellext::custom_state", "Custom State Handler registered with cookie: {}", cookie);
        }

        Ok(())
    }

    pub fn init_and_start_view_online_handler(&mut self) -> Result<()> {
        tracing::info!(target: "shellext::context_menu", "Initializing Shell Services (View Online Handler)...");

        unsafe {
            // Create and register the class factory
            let factory: IClassFactory =
                CrExplorerCommandFactory::new(self.drive_manager.clone()).into();

            let cookie = CoRegisterClassObject(
                &CLSID_EXPLORER_COMMAND,
                &factory,
                CLSCTX_LOCAL_SERVER,
                REGCLS_MULTIPLEUSE,
            )?;

            self.cookies.push(cookie);
            tracing::info!(target: "shellext::context_menu", "Context Menu Handler registered with cookie: {}", cookie);
        }

        Ok(())
    }

    pub fn run_message_loop(&self) -> Result<()> {
        tracing::info!(target: "shellext::context_menu", "Context Menu Handler is running. Press Ctrl+C to exit...");

        // Keep the thread alive to handle COM requests
        // In the C++ version, they use CoWaitForMultipleHandles
        // We'll use a simple approach - create a dummy event handle
        unsafe {
            // Use INVALID_HANDLE_VALUE as a dummy handle for CoWaitForMultipleHandles
            // This keeps the COM message pump running
            let dymmyevent = CreateEventW(None, FALSE, FALSE, None)?;
            let index =
                CoWaitForMultipleHandles((COWAIT_DISPATCH_CALLS).0 as u32, u32::MAX, &[dymmyevent]);
            tracing::info!(target: "shellext::context_menu", "CoWaitForMultipleHandles index: {:?}", index);
        }

        Ok(())
    }
}

impl Drop for ShellServices {
    fn drop(&mut self) {
        tracing::info!(target: "shellext::context_menu", "Unregistering Shell Services...");
        unsafe {
            for cookie in &self.cookies {
                let _ = CoRevokeClassObject(*cookie);
            }
            CoUninitialize();
        }
    }
}
