// @generated automatically by Diesel CLI.
diesel::table! {
    file_metadata (id) {
        id -> BigInt,
        drive_id -> Text,
        is_folder -> Bool,
        local_path -> Text,
        created_at -> BigInt,
        updated_at -> BigInt,
        etag -> Text,
        metadata -> Text,
        props -> Nullable<Text>,
        permissions -> Text,
        shared -> Bool,
        size -> BigInt,
        conflict_state -> Nullable<Text>,
    }
}

diesel::table! {
    task_queue (id) {
        id -> Text,
        drive_id -> Text,
        task_type -> Text,
        local_path -> Text,
        status -> Text,
        progress -> Double,
        total_bytes -> BigInt,
        processed_bytes -> BigInt,
        priority -> Integer,
        custom_state -> Nullable<Text>,
        error -> Nullable<Text>,
        created_at -> BigInt,
        updated_at -> BigInt,
    }
}

diesel::table! {
    upload_sessions (id) {
        id -> Text,
        task_id -> Text,
        drive_id -> Text,
        local_path -> Text,
        remote_uri -> Text,
        file_size -> BigInt,
        chunk_size -> BigInt,
        policy_type -> Text,
        session_data -> Text,
        chunk_progress -> Text,
        encrypt_metadata -> Nullable<Text>,
        expires_at -> BigInt,
        created_at -> BigInt,
        updated_at -> BigInt,
    }
}

diesel::table! {
    drive_props (id) {
        id -> BigInt,
        drive_id -> Text,
        capacity -> Nullable<Text>,
        capacity_updated_at -> Nullable<BigInt>,
        storage_policies -> Nullable<Text>,
        storage_policies_updated_at -> Nullable<BigInt>,
        user_settings -> Nullable<Text>,
        user_settings_updated_at -> Nullable<BigInt>,
        created_at -> BigInt,
        updated_at -> BigInt,
    }
}
