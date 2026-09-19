pub mod shell_service {
    use std::sync::Arc;

    use crate::drive::manager::DriveManager;

    #[derive(Debug, Default)]
    pub struct ServiceHandle;

    impl ServiceHandle {
        pub fn wait_for_init(&mut self) -> anyhow::Result<()> {
            Ok(())
        }

        pub fn shutdown(&self) {}
    }

    pub fn init_and_start_service_task(_drive_manager: Arc<DriveManager>) -> ServiceHandle {
        tracing::info!(target: "shellext::shell_service", "Shell services are not available on this platform");
        ServiceHandle
    }
}
