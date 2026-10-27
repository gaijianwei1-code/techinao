<h1 align="center">Techinao</h1>

<p align="center">
  全能 AI 创作工作台：无限画布编排、AI 图片 / 视频 / 音频生成、参考图编辑、提示词库与素材沉淀，一个界面完成从灵感到成片
</p>

<p align="center">
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-AGPL--3.0-f97316?style=flat-square" alt="License"></a>
  <a href="https://www.docker.com/"><img src="https://img.shields.io/badge/Docker-ready-2496ed?style=flat-square&logo=docker&logoColor=white" alt="Docker ready"></a>
  <a href="https://nextjs.org/"><img src="https://img.shields.io/badge/Next.js-16-000000?style=flat-square&logo=nextdotjs" alt="Next.js"></a>
  <a href="https://go.dev/"><img src="https://go.dev/img/go-blue.svg" alt="Go"></a>
</p>

<p align="center">
  <img src="web/public/logo.svg" width="96" alt="techinao logo">
</p>

## 项目介绍

Techinao 是一款面向图片、视频、音频创作的 AI 工作台。它把画布编排、AI 生成、素材管理和提示词库放在同一个界面里，支持账号云同步，既能本地单人使用，也能部署成多人服务。

## 核心功能

- 无限画布：多画布项目管理、节点拖拽缩放、连线编排、小地图、撤销重做、导入导出、插件扩展
- Agent 创作助手：围绕选中节点和上游节点分析、规划并生成图片或视频，结果自动插回画布
- AI 图片：OpenAI 兼容接口生图 / 图生图 / 参考图编辑、流式接收、批量生成
- AI 视频：Seedance、Kling、Veo 等模型文生视频、图生视频、多镜头编排
- AI 音频：TTS 语音合成、多音色支持
- 导演台：3D 场景布置角色、全景环境和机位，机位画面直连画布图片节点
- 全景图：文生全景、参考图生成、本地 2:1 全景导入
- 生图工作台：多任务并发、历史结果管理、分类与素材复用
- 创作工作流：公开 / 个人模板、变量表单、结果自动进入生图历史
- 提示词库：内置数百个精选图片提示词案例
- 账号体系：登录注册、素材云同步、管理后台（模型渠道、用户、提示词、素材管理）

完整功能说明见 [功能介绍](docs/overview/features.md)

## 技术栈

- 前端：Next.js、React、TypeScript、Tailwind CSS、Ant Design、Zustand
- 后端：Go、Gin、GORM、SQLite
- 部署：Docker Compose

## 快速开始

```bash
git clone https://github.com/gaijianwei1-code/techinao.git
cd techinao
cp .env.example .env
# 修改默认账号密码等信息
docker compose -f docker-compose.local.yml up -d --build
```

运行后访问 `http://localhost:3030`

本地开发运行：

```bash
cp .env.example .env
go run .

# 另开一个终端窗口
cd web
bun install
bun run dev
```

## 文档

- [功能介绍](docs/overview/features.md)
- [快速开始](docs/overview/quick-start.md)
- [部署说明](docs/overview/docker.md)
- [画布节点操作手册](docs/canvas/canvas-node-manual.md)
- [画布快捷键](docs/canvas/canvas-shortcuts.md)
- [待办事项](docs/progress/todo.md)
- [后端数据库说明](docs/backend/backend-database.md)
- [接口响应约定](docs/backend/api-response.md)

## 致谢

本项目基于 [basketikun/infinite-canvas](https://github.com/basketikun/infinite-canvas) 二次开发而来，感谢原作者及社区贡献者。

## 开源协议

本项目使用 GNU Affero General Public License v3.0，见 [LICENSE](LICENSE)。
