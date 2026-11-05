# 管理员接收爬虫文件

基于 `custom/4.19.1-receive`，接收接口支持可选的 `name`：

```http
POST /api/v4/admin/file/receive
Authorization: Bearer <管理员访问令牌>
Content-Type: application/json

{
  "source_id": "1x2ga",
  "dst": "cloudreve://my/聚水潭/账号A/销售主题分析",
  "name": "2026-10-1_2026-10-31_销售主题分析.xlsx"
}
```

`dst` 仍是事先创建的管理员目录。`name` 省略或为空时保留来源文件名，兼容旧调用。
指定 `name` 时直接以该名称创建管理员文件记录，来源文件名不变，Blob 不搬运、不重复上传。
目标路径加锁、重名校验、配额和引用计数在原接收事务中完成；重名失败不覆盖，不遗留中间文件。
文件名不得包含路径、相对目录或 NUL/换行字符。成功时返回 `data.id` 新 ID 和 `data.path` 最终 URI。

只有管理员可以接收，OAuth 令牌需要 `Admin.Write` 和 `Files.Write` 及对应读权限。
原有来源类型、上传完成、回收站和配额限制继续适用。源文件不会自动删除，接收成功后可由原用户正常删除。
重试应使用首次成功返回的新 ID。升级 Cloudreve 服务后再启用调用 `name` 参数的后台版本。
