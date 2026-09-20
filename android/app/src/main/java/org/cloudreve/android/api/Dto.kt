package org.cloudreve.android.api

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class ApiResponse<T>(
    val code: Int = 0,
    val msg: String = "",
    val data: T? = null,
)

@Serializable
data class Token(
    @SerialName("access_token") val accessToken: String = "",
    @SerialName("refresh_token") val refreshToken: String = "",
    @SerialName("access_expires") val accessExpires: String = "",
    @SerialName("refresh_expires") val refreshExpires: String = "",
)

@Serializable
data class User(
    val id: String = "",
    val email: String = "",
    val nickname: String = "",
    @SerialName("user_name") val userName: String = "",
    val status: String = "",
    val avatar: String = "",
)

@Serializable
data class LoginResponse(
    val user: User = User(),
    val token: Token = Token(),
)

@Serializable
data class PasswordLoginRequest(
    val email: String,
    val password: String,
)

@Serializable
data class RefreshTokenRequest(
    @SerialName("refresh_token") val refreshToken: String,
)

@Serializable
data class RefreshTokenResponse(
    @SerialName("access_token") val accessToken: String = "",
    @SerialName("refresh_token") val refreshToken: String = "",
    @SerialName("access_expires") val accessExpires: String = "",
    @SerialName("refresh_expires") val refreshExpires: String = "",
)

@Serializable
data class FileObject(
    val type: Int = 0,
    val id: String = "",
    val name: String = "",
    val permission: String? = null,
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("updated_at") val updatedAt: String = "",
    val size: Long = 0,
    val metadata: Map<String, String>? = null,
    val path: String = "",
    val shared: Boolean? = null,
    val thumbnail: Boolean? = null,
) {
    val isFolder: Boolean get() = type == 1
}

@Serializable
data class Pagination(
    @SerialName("next_token") val nextToken: String? = null,
    @SerialName("page_size") val pageSize: Int = 0,
    @SerialName("is_cursor") val isCursor: Boolean = false,
)

@Serializable
data class ListFileResponse(
    val files: List<FileObject> = emptyList(),
    val pagination: Pagination = Pagination(),
    @SerialName("storage_policy") val storagePolicy: StoragePolicy? = null,
    val parent: FileObject? = null,
)

@Serializable
data class StoragePolicy(
    val id: String = "",
    val name: String = "",
    val type: String = "",
)

@Serializable
data class FileUrlRequest(
    val uris: List<String>,
    val download: Boolean = false,
)

@Serializable
data class FileUrlResponse(
    val urls: List<FileUrl> = emptyList(),
)

@Serializable
data class FileUrl(
    val url: String = "",
)

@Serializable
data class CreateFileRequest(
    val uri: String,
    val type: String,
    @SerialName("err_on_conflict") val errOnConflict: Boolean = false,
)

@Serializable
data class RenameFileRequest(
    val uri: String,
    @SerialName("new_name") val newName: String,
)

@Serializable
data class DeleteFileRequest(
    val uris: List<String>,
    val unlink: Boolean = false,
)

@Serializable
data class CreateUploadSessionRequest(
    val uri: String,
    val size: Long,
    @SerialName("last_modified") val lastModified: Long = 0,
    @SerialName("mime_type") val mimeType: String = "",
    @SerialName("policy_id") val policyId: String = "",
    @SerialName("entity_type") val entityType: String = "",
    val previous: String = "",
)

@Serializable
data class UploadSessionResponse(
    @SerialName("session_id") val sessionId: String = "",
    @SerialName("upload_id") val uploadId: String = "",
    @SerialName("chunk_size") val chunkSize: Long = 0,
    val expires: Long = 0,
    @SerialName("upload_urls") val uploadUrls: List<String>? = null,
    @SerialName("completeURL") val completeUrl: String = "",
    val uri: String = "",
    @SerialName("storage_policy") val storagePolicy: StoragePolicy? = null,
    @SerialName("callback_secret") val callbackSecret: String = "",
)

@Serializable
data class ThumbResponse(
    val url: String = "",
    val expires: String? = null,
)

@Serializable
data class ShareCreateRequest(
    val uri: String,
    val downloads: Int = 0,
    @SerialName("is_private") val isPrivate: Boolean = false,
    val password: String = "",
    val expire: Int = 0,
    @SerialName("share_view") val shareView: Boolean = false,
    @SerialName("show_readme") val showReadme: Boolean = false,
    @SerialName("allow_upload") val allowUpload: Boolean = false,
    @SerialName("allow_edit") val allowEdit: Boolean = false,
    @SerialName("preview_only") val previewOnly: Boolean = false,
    @SerialName("upload_only") val uploadOnly: Boolean = false,
)

@Serializable
data class SearchHit(
    val file: FileObject = FileObject(),
    val content: String = "",
)

@Serializable
data class SearchResponse(
    val hits: List<SearchHit> = emptyList(),
    val total: Long = 0,
)

@Serializable
data class SiteConfig(
    val authn: Boolean = false,
    @SerialName("register_enabled") val registerEnabled: Boolean = false,
    @SerialName("invitation_code") val invitationCode: Boolean = false,
    @SerialName("tos_url") val tosUrl: String = "",
    @SerialName("privacy_policy_url") val privacyPolicyUrl: String = "",
)

@Serializable
data class TaskItem(
    val id: String = "",
    val status: String = "",
    val type: String = "",
    val error: String? = null,
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class TaskListResponse(
    val tasks: List<TaskItem> = emptyList(),
)

@Serializable
data class OAuthTokenResponse(
    @SerialName("access_token") val accessToken: String = "",
    @SerialName("refresh_token") val refreshToken: String = "",
    @SerialName("expires_in") val expiresIn: Long = 0,
    @SerialName("token_type") val tokenType: String = "",
)

@Serializable
data class UserInfoResponse(
    val sub: String = "",
    val name: String = "",
    @SerialName("preferred_username") val preferredUsername: String = "",
    val email: String = "",
)
