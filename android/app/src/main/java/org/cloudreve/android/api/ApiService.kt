package org.cloudreve.android.api

import okhttp3.RequestBody
import okhttp3.ResponseBody
import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.HTTP
import retrofit2.http.POST
import retrofit2.http.PUT
import retrofit2.http.Path
import retrofit2.http.Query
import retrofit2.http.Url

interface ApiService {

    @POST("api/v4/session/token")
    suspend fun login(@Body request: PasswordLoginRequest): Response<ApiResponse<LoginResponse>>

    @POST("api/v4/session/token/refresh")
    suspend fun refreshToken(@Body request: RefreshTokenRequest): Response<ApiResponse<RefreshTokenResponse>>

    @DELETE("api/v4/session/token")
    suspend fun signOut(@Body request: RefreshTokenRequest): Response<ApiResponse<Unit>>

    @GET("api/v4/site/config/login")
    suspend fun siteConfig(): Response<ApiResponse<SiteConfig>>

    @GET("api/v4/file")
    suspend fun listFiles(
        @Query("uri") uri: String,
        @Query("page_size") pageSize: Int = 50,
        @Query("next_page_token") nextPageToken: String? = null,
    ): Response<ApiResponse<ListFileResponse>>

    @POST("api/v4/file/url")
    suspend fun fileUrls(@Body request: FileUrlRequest): Response<ApiResponse<FileUrlResponse>>

    @POST("api/v4/file/create")
    suspend fun createFile(@Body request: CreateFileRequest): Response<ApiResponse<Unit>>

    @POST("api/v4/file/rename")
    suspend fun renameFile(@Body request: RenameFileRequest): Response<ApiResponse<Unit>>

    @HTTP(method = "DELETE", path = "api/v4/file", hasBody = true)
    suspend fun deleteFiles(@Body request: DeleteFileRequest): Response<ApiResponse<Unit>>

    @PUT("api/v4/file/upload")
    suspend fun createUploadSession(@Body request: CreateUploadSessionRequest): Response<ApiResponse<UploadSessionResponse>>

    // Chunk targets are fully resolved URLs returned in upload_urls —
    // same-origin for local policies, presigned for remote policies.
    @POST
    suspend fun uploadChunk(@Url url: String, @Body body: RequestBody): Response<ResponseBody>

    @PUT
    suspend fun uploadChunkPut(@Url url: String, @Body body: RequestBody): Response<ResponseBody>

    @POST
    suspend fun completeUpload(@Url url: String): Response<ResponseBody>

    @GET
    suspend fun download(@Url url: String): Response<ResponseBody>

    @GET("api/v4/file/thumb")
    suspend fun thumb(@Query("uri") uri: String): Response<ApiResponse<ThumbResponse>>

    @GET("api/v4/file/search")
    suspend fun searchFiles(
        @Query("query") query: String,
        @Query("offset") offset: Int = 0,
    ): Response<ApiResponse<SearchResponse>>

    @GET("api/v4/workflow")
    suspend fun listTasks(
        @Query("page_size") pageSize: Int = 20,
        @Query("category") category: String = "general",
    ): Response<ApiResponse<TaskListResponse>>

    @PUT("api/v4/share")
    suspend fun createShare(@Body request: ShareCreateRequest): Response<ApiResponse<String>>
}
