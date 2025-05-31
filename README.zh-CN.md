<h1 align="center">
  <img src="https://codeberg.org/repo-avatars/b9a8c82b56a5f8f466f3731164f71ed961ca971df89c6a1ceac618e3b5062050" alt="PixivFE logo" width="150" />
  <br />
  PixivFE
  <br />
  <a href="https://gitlab.com/pixivfe/PixivFE/-/commits/v3"><img alt="Pipeline status on GitLab" src="https://gitlab.com/pixivfe/PixivFE/badges/v3/pipeline.svg" /></a>
  <a href="https://crowdin.com/project/pixivfe" rel="nofollow"><img src="https://badges.crowdin.net/pixivfe/localized.svg" alt="Localization percentage on Crowdin" /></a>
</h1>

<!-- Relative paths are used here for PR -->
<p align="center">
  <b>
    <a href="./README.md">English</a> |
    <a href="./README.zh-CN.md">简体中文</a>
  </b>
</p>

PixivFE（全称 _Pixiv FrontEnd_）是一个开源的、可自托管的 [pixiv](https://zh.wikipedia.org/wiki/Pixiv) 替代前端。

您可以立即通过我们的[官方公共实例](https://pixiv.perennialte.ch/)体验，或查看[公共实例列表](https://pixivfe-docs.pages.dev/instance-list/)。

欢迎阅读[项目文档](https://pixivfe-docs.pages.dev/)获取安装指南和更多信息，亦可参阅 [roadmap](https://pixivfe-docs.pages.dev/dev/roadmap/) 和 [scope](https://pixivfe-docs.pages.dev/dev/scope/)。

我们使用 [WeKan (看板)](https://kanban.adminforge.de/b/ZDTHNygpkXerQRgcq/pixivfe) 管理项目。

## 为何选择 PixivFE？

- **匿名浏览**：无需 pixiv 账号即可匿名访问内容，解除**所有**浏览限制。
- **隐私优先**：所有数据处理**均在服务器端完成**，客户端仅保留前端交互，彻底隔绝与 pixiv 及其第三方服务（如 Google Analytics）的直接接触。
- **轻量现代**：PixivFE 遵循[**渐进增强**](https://developer.mozilla.org/zh-CN/docs/Glossary/Progressive_Enhancement)设计理念，确保无 JavaScript 环境下核心功能依然可用。当用户启用 JavaScript 时，将启用异步加载等增强功能，实现更快的页面导航和无需整页刷新的流畅交互体验。我们的界面轻量现代，与 pixiv 原站臃肿的前端形成鲜明对比，最大限度降低浏览干扰。
- **开放透明**：PixivFE 为**自由软件**，代码公开、开发透明，支持自由修改和二次开发。

PixivFE 致力于打造自由、隐私、无障碍的浏览体验。如果您认同这些理念，欢迎[立即体验](https://pixivfe-docs.pages.dev/instance-list/)——更推荐[自行部署](https://pixivfe-docs.pages.dev/hosting/)！

## 项目定位

- ❌ 非 _pixiv_ 官方产品
- ❌ 非内容爬虫工具（请勿滥用）
- ❌ 非全功能客户端（部分功能尚未实现，详见 [roadmap](https://pixivfe-docs.pages.dev/dev/roadmap/)）

## 快速部署

支持通过 [Docker](https://pixivfe-docs.pages.dev/hosting/hosting-pixivfe/#docker) 或[源码编译](https://pixivfe-docs.pages.dev/hosting/hosting-pixivfe/#binary) 两种方式部署。

## 开发指南

使用构建工具：`./build.sh help`

下面是一些构建依赖项，您可以选择性安装。

- [Go 1.24+](https://go.dev/doc/install)
- [Tailwind CSS CLI](https://github.com/tailwindlabs/tailwindcss/releases/latest)
- [jq](https://jqlang.github.io/jq/)（可选，用于构建国际化文件）
- [Crowdin CLI](./doc/dev/features/i18n.md)（可选，用于构建国际化文件）

安装 Tailwind CSS CLI：

```bash
curl -qsLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64
chmod +x tailwindcss-linux-x64
mv tailwindcss-linux-x64 ~/.local/bin/tailwindcss

# 或者作为单个命令
curl -qsLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64 \
  && chmod +x tailwindcss-linux-x64 \
  && mv tailwindcss-linux-x64 ~/.local/bin/tailwindcss
```

运行项目：

```bash
# 克隆仓库
git clone https://codeberg.org/PixivFE/PixivFE.git && cd PixivFE

# 开发模式运行（模板热更新）
PIXIVFE_DEV=true <其他环境变量> ./build.sh run

# 另启终端监听 CSS 变更
tailwindcss -i assets/css/tailwind-style_source.css -o assets/css/tailwind-style.css --watch --minify
```

### Nix

如果您的机器上已安装 [Nix](https://wiki.archlinux.org/title/Nix)，可以通过运行以下命令启动[开发环境](https://nix.dev/tutorials/first-steps/declarative-shell.html)：

```
$ nix-shell

...
Tailwind CSS daemon is running with PID 68596
You may start running PixivFE now. Example: ./build.sh run

$ # 欢迎使用 Nix shell
```
### 源码

**注：本项目托管于两个同步的仓库**:

- [Codeberg](https://codeberg.org/PixivFE/PixivFE) 为官方主仓库，接受 issue 和 PR
- [GitLab](https://gitlab.com/pixivfe/PixivFE) 用于 CI/CD 流水线

两仓库保持实时同步，提交至任一仓库将自动同步至另一仓库。

## 获取支持

如需帮助或提交反馈：

- 加入我们的 [Matrix 聊天室](https://matrix.to/#/#pixivfe:4d2.org)
- 使用 [Issue Tracker](https://codeberg.org/PixivFE/PixivFE/issues) 报告 Bug
- 联系 [VnPower](https://loang.net/~vnpower/me#contact)

## 开源协议

PixivFE 为自由软件，采用 [AGPLv3](https://www.gnu.org/licenses/agpl-3.0.zh-cn.html) 协议授权。
