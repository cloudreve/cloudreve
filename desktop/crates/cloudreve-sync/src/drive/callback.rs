use std::{sync::Arc, time::Duration};

use crate::{
    cfapi::{
        error::{CResult, CloudErrorKind},
        filter::{Request, SyncFilter, info, ticket},
        placeholder_file::PlaceholderFile,
    },
    drive::{
        commands::MountCommand,
        sync::{cloud_file_to_metadata_entry, cloud_file_to_placeholder, is_symbolic_link},
    },
    inventory::{InventoryDb, MetadataEntry},
};
use tokio::sync::mpsc;
use uuid::Uuid;

#[derive(Clone)]
pub struct CallbackHandler {
    command_tx: mpsc::UnboundedSender<MountCommand>,
    id: String,
    inventory: Arc<InventoryDb>,
}

impl CallbackHandler {
    pub fn new(
        command_tx: mpsc::UnboundedSender<MountCommand>,
        id: String,
        inventory: Arc<InventoryDb>,
    ) -> Self {
        Self {
            command_tx,
            id: id,
            inventory: inventory,
        }
    }

    pub async fn sleep(&self) {
        tokio::time::sleep(Duration::from_secs(10)).await;
    }
}

impl SyncFilter for CallbackHandler {
    fn fetch_data(
        &self,
        request: Request,
        ticket: ticket::FetchData,
        info: info::FetchData,
    ) -> crate::cfapi::error::CResult<()> {
        let (response_tx, response_rx) = tokio::sync::oneshot::channel();
        let command = MountCommand::FetchData {
            path: request.path().to_path_buf(),
            ticket,
            range: info.required_file_range(),
            response: response_tx,
        };
        if let Err(e) = self.command_tx.send(command) {
            tracing::error!(target: "drive::mounts", id = %self.id, error = %e, "Failed to send FetchData command");
            return Err(CloudErrorKind::NotSupported);
        }

        match response_rx.blocking_recv() {
            Ok(Ok(())) => Ok(()),
            _ => Err(CloudErrorKind::Unsuccessful),
        }
    }

    fn deleted(&self, request: Request, _info: info::Deleted) {
        tracing::debug!(target: "drive::mounts", id = %self.id, path = %request.path().display(), "Deleted");
    }

    fn delete(&self, request: Request, ticket: ticket::Delete, _info: info::Delete) -> CResult<()> {
        tracing::debug!(target: "drive::mounts", id = %self.id, path = %request.path().display(), "Delete");
       let _ = ticket.pass();
        Ok(())
    }

    fn rename(&self, request: Request, ticket: ticket::Rename, info: info::Rename) -> CResult<()> {
        let src = request.path();
        let dest = info.target_path();
        tracing::debug!(target: "drive::mounts", id = %self.id, source_path = %src.display(), target_path = %dest.display(), "Rename");
        let (response_tx, response_rx) = tokio::sync::oneshot::channel();
        let command = MountCommand::Rename {
            source: src.to_path_buf(),
            target: dest.to_path_buf(),
            response: response_tx,
        };
        if let Err(e) = self.command_tx.send(command) {
            tracing::error!(target: "drive::mounts", id = %self.id, error = %e, "Failed to send rename command");
            return Err(CloudErrorKind::NotSupported);
        }

        match response_rx.blocking_recv() {
            Ok(Ok(())) => {
                let _ = ticket.pass();
                Ok(())
            }
            _ => Err(CloudErrorKind::Unsuccessful),
        }
        // TODO: delete sometimes trigger rename callback
    }

    fn fetch_placeholders(
        &self,
        request: Request,
        ticket: ticket::FetchPlaceholders,
        _info: info::FetchPlaceholders,
    ) -> CResult<()> {
        let (response_tx, response_rx) = tokio::sync::oneshot::channel();
        let command = MountCommand::FetchPlaceholders {
            path: request.path().to_path_buf(),
            response: response_tx,
        };
        if let Err(e) = self.command_tx.send(command) {
            tracing::error!(target: "drive::mounts", id = %self.id, error = %e, "Failed to send FetchPlaceholders command");
            return Err(CloudErrorKind::NotSupported);
        }

        match response_rx.blocking_recv() {
            Ok(Ok(files)) => {
                tracing::debug!(target: "drive::mounts", id = %self.id, files = %files.files.len(), "Received placeholders");
                let mut placeholders = files.files.iter()
                    .filter(|file| !is_symbolic_link(file))
                    .map(|file| cloud_file_to_placeholder(file, &files.local_path, &files.remote_path))
                    .filter_map(|result|{
                        if result.is_ok() {
                            Some(result.unwrap())
                        } else {
                            tracing::error!(target: "drive::mounts", id = %self.id, error = %result.unwrap_err(), "Failed to convert cloud file to placeholder");
                            None
                        }
                    })
                    .collect::<Vec<PlaceholderFile>>();
                if let Err(e) = ticket.pass_with_placeholder(&mut placeholders) {
                    tracing::error!(target: "drive::mounts", id = %self.id, error = %e, "Failed to pass placeholders");
                    return Err(CloudErrorKind::Unsuccessful);
                }
                tracing::debug!(target: "drive::mounts", id = %self.id, placeholders = %placeholders.len(), "Passed placeholders");

                // Insert placeholders into inventory
                let drive_id = Uuid::parse_str(&self.id)
                    .unwrap_or_else(|e| {
                        tracing::error!(target: "drive::mounts", id = %self.id, error = %e, "Failed to parse drive ID");
                        return Uuid::new_v4();
                    });
                let entries = files
                    .files
                    .iter()
                    .filter_map(|f| {
                        cloud_file_to_metadata_entry(f, &drive_id, &files.local_path).map_err(|e| {
                            tracing::error!(target: "drive::mounts", id = %self.id, error = %e, "Failed to convert cloud file to metadata entry");
                        }).ok()
                    })
                    .collect::<Vec<MetadataEntry>>();
                if let Err(e) = self.inventory.batch_insert(&entries) {
                    tracing::error!(target: "drive::mounts", id = %self.id, error = ?e, "Failed to insert placeholders into inventory");
                }
                return Ok(());
            }
            _ => {}
        }

        Err(CloudErrorKind::Unsuccessful)
    }

    fn closed(&self, request: Request, info: info::Closed) {
        tracing::debug!(target: "drive::mounts", id = %self.id, path = %request.path().display(), deleted = %info.deleted(), "Closed");
    }

    fn cancel_fetch_data(&self, _request: Request, _info: info::CancelFetchData) {
        tracing::debug!(target: "drive::mounts", id = %self.id, "CancelFetchData");
    }

    fn validate_data(
        &self,
        _request: Request,
        _ticket: ticket::ValidateData,
        _info: info::ValidateData,
    ) -> CResult<()> {
        tracing::debug!(target: "drive::mounts", id = %self.id, "ValidateData");
        Err(CloudErrorKind::NotSupported)
    }

    fn cancel_fetch_placeholders(&self, _request: Request, _info: info::CancelFetchPlaceholders) {
        tracing::debug!(target: "drive::mounts", id = %self.id, "CancelFetchPlaceholders");
    }

    fn opened(&self, request: Request, _info: info::Opened) {
        tracing::debug!(target: "drive::mounts", id = %self.id, path = %request.path().display(), "Opened");
    }

    fn dehydrate(
        &self,
        _request: Request,
        _ticket: ticket::Dehydrate,
        info: info::Dehydrate,
    ) -> CResult<()> {
        tracing::debug!(
            target: "drive::mounts",
            id = %self.id,
            reason = ?info.reason(),
            "Dehydrate"
        );
        Err(CloudErrorKind::NotSupported)
    }

    fn dehydrated(&self, _request: Request, info: info::Dehydrated) {
        tracing::debug!(
            target: "drive::mounts",
            id = %self.id,
            reason = ?info.reason(),
            "Dehydrated"
        );
    }

    fn renamed(&self, request: Request, info: info::Renamed) {
        let dest = request.path();
        tracing::debug!(target: "drive::mounts", id = %self.id, dest_path = %dest.display(), "Renamed");
        let command: MountCommand = MountCommand::Renamed {
            source: info.source_path(),
            destination: dest,
        };
        if let Err(e) = self.command_tx.send(command) {
            tracing::error!(target: "drive::mounts", id = %self.id, error = %e, "Failed to send Renamed command");
            return;
        }
    }
}
