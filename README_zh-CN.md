<h1 align="center">
  <br>
  Cloudreve — 社区维护分支
  <br>
</h1>
<h4 align="center">自托管文件管理与分享平台 — 完全开源，持续维护中。</h4>

> 本仓库是 [cloudreve/cloudreve](https://github.com/cloudreve/cloudreve) 的积极维护分支。
> 所有原始工作归属 Cloudreve 原作者（cloudreve.org）。由于上游开发放缓，本分支将项目延续为
> **完整、完全开源的发行版**：后端、Web 前端、Windows/macOS/Linux 桌面客户端以及原生 Android
> 应用，并将所有 “Pro” 级功能以自由软件方式重新实现。署名说明见 [NOTICE](NOTICE)。

## 与上游的差异

- **无 Pro 分层。** 已移除付费引导 UI（`ProChip`/`ProDialog`）；Pro 级功能以开源方式重新实现：
  分享协作（可上传/可编辑/仅预览/投递箱分享）、OIDC SSO、委派管理员已落地。
- **上游 issue 积压已分类修复。** 137 个迁移 issue 全部在本仓库跟踪，约 70% 已关闭，包括
  WebDAV 挂载/只读/冲突处理、上传卡死、回收站保护、SQLite WAL、MySQL `parseTime`、
  unix socket 迁移等。
- **在上游修复之上的安全加固。** OAuth 公共客户端不再内置硬编码密钥（仅 PKCE，符合 RFC 8252）；
  离线下载 URL 做 SSRF 校验；委派管理员操作记入审计日志；认证端点限流。
- **单仓 monorepo。** 后端、前端、桌面端、Android 端同仓管理，无 submodule。
- **测试与 CI 真实运行。** 每个 PR 执行后端测试、前端类型检查/构建、桌面端三平台构建。

## 仓库结构

```
.                    Go 后端 — Gin + ent ORM（SQLite/MySQL/PostgreSQL）
frontend/            Web 前端 — React + TypeScript + Vite + MUI（已内嵌，无 submodule）
desktop/             桌面客户端 — Tauri/Rust 同步引擎（当前为 Windows cfapi；
                     macOS/Linux 待接入，见路线图）
android/             原生 Android 客户端 — Kotlin + Jetpack Compose（脚手架阶段）
.github/workflows/   CI（后端、前端、桌面端矩阵）+ 发布流水线
```

## 功能

- 存储端：本地、远程节点、S3 兼容、OneDrive、OSS、COS、Qiniu、Upyun、KS3、OBS。
- 客户端与存储端直传；分块、断点续传、并行上传。
- 离线下载：aria2、qBittorrent、**yt-dlp** 三种提供方，多节点、按节点配置，
  用户组级并发/体积配额。
- 分享链接：过期时间、仅上传投递箱、在线编辑、仅预览、匿名上传、IP 限制访问。
- 压缩包解压/打包、媒体元数据提取、元数据/标签检索。
- 全存储端 WebDAV（本分支修复了只读用户组的权限执行问题）。
- SSO：通用 OIDC 接入（授权码 + nonce、JWKS 校验、自动开户）、OAuth 公共客户端 PKCE、
  Passkey、TOTP 两步验证。
- 多用户多用户组；管理员任务列表支持 CIDR 创建者 IP 过滤；按用户回收站保留期；
  按用户组离线下载配额。
- 预览：图片（缩略图渐进加载到原图）、视频、音频、ePub、Markdown、图表、Office 文档、3D 模型。
- PWA、深色模式、多语言、主题自定义、自定义 HTML 注入。

## 从源码构建

依赖：Go ≥ 1.24、Node ≥ 20 + Yarn、（桌面端）Rust + Tauri 平台依赖。

```bash
# 前端
cd frontend && yarn install
NODE_OPTIONS=--max-old-space-size=6144 yarn build   # 产物由 Go embed 打包

# 后端（仓库根目录）— 二进制同时托管前端与 API，监听 :5212
go build -o cloudreve .
./cloudreve
```

桌面客户端见 `desktop/CLAUDE.md`（`cargo tauri build`，Windows 优先）。

## 开发

```bash
go build ./... && go vet ./... && go test ./...          # 后端门禁
cd frontend && yarn tsc --noEmit && yarn build           # 前端门禁
```

`docker-compose.dev.yml` 可启动 postgres + redis + 源码构建的后端；`yarn dev` 提供前端热更新。
PR 一律走功能分支，禁止直接推送 `master`，合并前必须通过全部 CI。

## 状态一览

| 领域 | 状态 |
|---|---|
| 后端 / 前端 | 稳定 — 4.19.1 基线，CI 全绿 |
| 上游 issue | 137 个迁移 issue 约 70% 已关闭；其余为大型功能、Pro 表面或依赖设备 |
| 代码健康 | desloppify 严格分 77.1（原 18.9）；73 项评审全部处置 |
| 桌面客户端 | Windows 可用（cfapi 同步 + 外壳集成）；macOS/Linux 计划中 |
| Android 客户端 | 脚手架完成 — Kotlin/Compose 骨架，见 [ROADMAP.md](ROADMAP.md) Phase E |
| Pro 免费化 | 分享协作 ✓、OIDC SSO ✓、委派管理员 ✓；存储策略迁移、VAS/计费、审计界面进行中 |

完整计划与已知限制见 [ROADMAP.md](ROADMAP.md)；issue 跟踪器中每项均有真实状态说明。

## 安全

请通过本仓库 GitHub 的 “Report a vulnerability” 私下报告漏洞，勿开公开 issue。
上游已公布的 16 个 GHSA 在本基线均已修复；新增改动合并前均经 SSRF、路径穿越、
进程执行与会话熵审查。

## 致谢

Cloudreve 由 **Aaron Liu 及 Cloudreve 贡献者** 创建（cloudreve.org）。本分支是在同一
GPL-3.0 许可下的独立延续 — 署名而非背书。完整声明见 [NOTICE](NOTICE)。

## 许可证

[GPL-3.0](LICENSE) — 与上游一致，贡献按同一许可提交。
