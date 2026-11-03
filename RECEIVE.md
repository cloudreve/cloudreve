# 管理员接收爬虫文件

基于官方 `4.19.1`，分支为 `custom/4.19.1-receive`。

## 接收接口

```http
POST /api/v4/admin/file/receive
Authorization: Bearer <管理员访问令牌>
Content-Type: application/json

{
  "source_id": "爬虫上传完成后得到的文件ID",
  "dst": "cloudreve://my/收件箱"
}
```

目标目录需事先创建。返回 `code: 0` 时，`data.id` 是管理员自己的新文件 ID，`data.path` 是新文件 URI。

- 仅管理员可调用，目标目录只能属于当前管理员。
- OAuth 令牌需具备 `Admin.Write` 和 `Files.Write`（包含对应读权限）。
- 支持已经完成上传的单个普通文件；不接收文件夹、快捷方式、回收站文件或仍在上传的当前版本。
- 文件内容仅上传一次。接收时创建新的文件记录，复用原 Blob、已有完整版本及缩略图，并增加引用计数。
- 爬虫在确认接收成功后，可以通过 Cloudreve 正常删除自己的文件；管理员文件继续保留。
- 原始物理存储路径不变，接收不会把数据搬到另一个存储策略。双方分别计算网盘配额。
- 目标重名会失败，不覆盖原文件；重试同一请求时应保留并使用首次成功返回的新文件 ID。
- 接口不会自动删除源文件。只保存源 ID 或分享快捷方式，不构成独立接收。

## 本地源码与 GitHub

源码位置为 `D:\project\cloudreve`。`origin` 指向 `https://github.com/duminqing/cloudreve.git`，`upstream` 指向官方仓库。

```powershell
Set-Location D:\project\cloudreve
git switch custom/4.19.1-receive
# 修改、测试后提交
git add <本次修改的文件>
git commit -m "描述这次修改"
git push origin custom/4.19.1-receive
```

## 从 GitHub 源码构建并更新现有服务

在已提交并推送代码后运行：

```powershell
pwsh -File D:\project\cloudreve\deploy\Update-Cloudreve.ps1 -Revision HEAD
```

脚本会解析到完整提交 SHA，再让 Docker 从该 GitHub 提交下载源码及固定的前端子模块，通过 `Dockerfile.receive` 构建镜像。镜像构建会运行接收功能的数据库集成测试和管理员服务测试。

镜像构建完成后，脚本只停止 Cloudreve，备份 PostgreSQL 的 `cloudreve` 数据库、`cloudreve_data` 卷和现有部署配置，然后写入 `D:\docker\server\cloudreve\compose.custom.yaml`，重建 Cloudreve 容器并等待健康检查。数据库和 Redis 服务继续运行。数据库结构没有新增表或列。

配置、密码和实际文件留在原部署目录及 Docker 卷内，均不提交 GitHub。默认分支 `master` 保留官方同步用途；不要直接把未经适配的新官方版本合进当前部署分支。

回退镜像：

```powershell
pwsh -File D:\project\cloudreve\deploy\Rollback-Cloudreve.ps1 -BackupDirectory 'D:\docker\server\cloudreve\backup\receive-实际时间-提交号'
```

回退保留数据库和网盘数据，已接收的文件仍是标准 Cloudreve 文件记录。完整数据恢复另需同时恢复备份中的数据库与文件卷。

日常管理自定义部署时，带上两个 Compose 文件：

```powershell
Set-Location D:\docker\server\cloudreve
docker compose -f compose.yaml -f compose.custom.yaml up -d --wait
```
