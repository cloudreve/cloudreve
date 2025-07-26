[中文版本](https://github.com/cloudreve/Cloudreve/blob/master/README_zh-CN.md)

<h1 align="center">
  <br>
  <a href="https://cloudreve.org/" alt="logo" ><img src="https://raw.githubusercontent.com/cloudreve/frontend/master/public/static/img/logo192.png" width="150"/></a>
  <br>
  Cloudreve
  <br>
</h1>
<h4 align="center">Self-hosted file management system with multi-cloud support.</h4>

<p align="center">
  <a href="https://dev.azure.com/abslantliu/Cloudreve/_build?definitionId=6">
    <img src="https://dev.azure.com/abslantliu/Cloudreve/_apis/build/status%2Fcloudreve.Cloudreve?branchName=refs%2Fpull%2F2481%2Fmerge"
         alt="Azure pipelines">
  </a>
  <a href="https://github.com/cloudreve/Cloudreve/releases">
    <img src="https://img.shields.io/github/v/release/cloudreve/Cloudreve?include_prereleases" />
  </a>
  <a href="https://hub.docker.com/r/cloudreve/cloudreve">
     <img src="https://img.shields.io/docker/image-size/cloudreve/cloudreve"/>
  </a>
  <a href="https://hub.docker.com/r/cloudreve/cloudreve">
  <img alt="Docker Pulls" src="https://img.shields.io/docker/pulls/cloudreve/cloudreve" />
  </a>
</p>
<p align="center">
  <a href="https://cloudreve.org">Homepage</a> •
  <a href="https://demo.cloudreve.org">Try it</a> •
  <a href="https://github.com/cloudreve/cloudreve/discussions">Discussion</a> •
  <a href="https://docs.cloudreve.org">Documents</a> •
  <a href="https://github.com/cloudreve/Cloudreve/releases">Download</a> •
  <a href="https://t.me/cloudreve_official">Telegram</a> •
  <a href="https://discord.com/invite/WTpMFpZT76">Discord</a>
</p>

![Screenshot](https://raw.githubusercontent.com/cloudreve/docs/master/images/homepage.png)

## :sparkles: Features

- :cloud: Support storing files into Local, Remote node, OneDrive, S3 compatible API, Qiniu, Aliyun OSS, Tencent COS, Upyun.
- :outbox_tray: Upload/Download in directly transmission from client to storage providers.
- 💾 Integrate with Aria2/qBittorrent to download files in background, use multiple download nodes to share the load.
- 📚 Compress/Extract files, download files in batch.
- 💻 WebDAV support covering all storage providers.
- :zap:Drag&Drop to upload files or folders, with resumable upload support.
- :card_file_box: Extract media metadata from files, search files by metadata or tags.
- :family_woman_girl_boy: Multi-users with multi-groups.
- :link: Create share links for files and folders with expiration date.
- :eye_speech_bubble: Preview videos, images, audios, ePub files online; edit texts, diagrams, Markdown, images, Office documents online.
- :art: Customize theme colors, dark mode, PWA application, SPA, i18n.
- :rocket: All-in-one packaging, with all features out of the box.
- 🌈 ... ...

## :hammer_and_wrench: Deploy

To deploy Cloudreve, you can refer to [Getting started](https://docs.cloudreve.org/overview/quickstart) for a quick local deployment to test.

When you're ready to deploy Cloudreve to a production environment, you can refer to [Deploy](https://docs.cloudreve.org/overview/deploy/) for a complete deployment.

## :gear: Build

Please refer to [Build](https://docs.cloudreve.org/overview/build/) for how to build Cloudreve from source code.

## :rocket: Contributing

If you're interested in contributing to Cloudreve, please refer to [Contributing](https://docs.cloudreve.org/api/contributing/) for how to contribute to Cloudreve.

## :alembic: Stacks

- [Go](https://golang.org/) + [Gin](https://github.com/gin-gonic/gin) + [ent](https://github.com/ent/ent)
- [React](https://github.com/facebook/react) + [Redux](https://github.com/reduxjs/redux) + [Material-UI](https://github.com/mui-org/material-ui)

## :scroll: License

GPL V3
