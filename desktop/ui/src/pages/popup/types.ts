export interface DriveConfig {
  id: string;
  name: string;
  instance_url: string;
  sync_path: string;
  icon_path?: string;
}

export interface TaskProgress {
  task_id: string;
  kind: "Upload" | "Download";
  local_path: string;
  progress: number;
  processed_bytes?: number;
  total_bytes?: number;
  speed_bytes_per_sec: number;
  eta_seconds?: number;
}

export interface TaskRecord {
  id: string;
  drive_id: string;
  task_type: string;
  local_path: string;
  status: "Pending" | "Running" | "Completed" | "Failed" | "Cancelled";
  progress: number;
  total_bytes: number;
  processed_bytes: number;
  error?: string;
  created_at: number;
  updated_at: number;
}

export interface TaskWithProgress extends TaskRecord {
  live_progress?: TaskProgress;
}

export interface StatusSummary {
  drives: DriveConfig[];
  active_tasks: TaskWithProgress[];
  finished_tasks: TaskRecord[];
}

export interface FileIconResponse {
  data: string; // Base64 encoded RGBA pixel data
  width: number;
  height: number;
}
