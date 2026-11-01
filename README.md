# vben-admin · GoFrame + Vue Vben Admin 后台管理系统

基于 **GoFrame v2** 与 **Vue Vben Admin v5** 的后台管理脚手架：后端负责认证、用户层级与数据权限、后端下发动态路由、日志与在线会话；前端基于 Vben Admin（Element Plus 版本）实现登录、用户管理、路由管理、日志管理与在线用户监控。

- 仓库地址：<https://github.com/2213473846/demo>
- 后端 API 文档（启动后）：<http://127.0.0.1:8000/swagger>
- 前端默认账号：`admin / 123456`

---

## 一、技术栈

| 层 | 技术 | 说明 |
| --- | --- | --- |
| 后端 | Go + GoFrame v2 (v2.10.4) | HTTP 服务、ORM、配置、日志 |
| 数据库 | MySQL 8.0（5.7+ 亦可） | 用户/菜单/授权/日志表 |
| 缓存 | Redis 6+ | accessToken、refreshToken、在线会话 |
| 前端 | Vue 3 + Vite + Element Plus + Vben Admin v5 | monorepo（pnpm workspace + turbo） |
| 认证 | Bearer Token + HttpOnly Refresh Cookie | accessToken 2 小时，refreshToken 7 天 |

## 二、功能特性

- **认证**：登录签发 accessToken（Redis 存储，2h）与 refreshToken（HttpOnly Cookie，7d），支持无感刷新、登出、token 强制作废。
- **用户/管理员管理**：树形层级（`pid`），数据权限隔离（`super_admin` 可见全部，`admin` 仅可见自己的子孙用户），软删除、回收站、恢复、彻底删除。
- **路由（菜单）管理**：后端下发路由树（Vben `accessMode: backend`），支持目录/菜单、图标、排序、隐藏、缓存、公开页；按钮级权限码由 `/auth/codes` 下发。
- **日志管理**：登录日志（成功/失败、IP、浏览器、操作系统）与操作日志（用户、菜单等写操作埋点）。
- **系统监控**：在线用户列表（会话续期、强退下线）。
- **统一响应**：`{code, data, error, message}`，401/403 与前端刷新流程对齐。

## 三、目录结构

```text
.
├── api/                     # 接口定义(Req/Res), 按业务域划分
├── internal/
│   ├── cmd/                 # 服务入口与路由注册
│   ├── consts/              # 角色、错误码、Redis key、白名单
│   ├── controller/          # 控制器(由 gf gen ctrl 生成)
│   ├── dao/                 # 数据访问(自动生成, 勿手改)
│   ├── model/
│   │   ├── do/              # 写操作对象(自动生成)
│   │   └── entity/          # 表实体(自动生成)
│   └── service/             # 业务逻辑
│       ├── auth/ boot/ identity/ log/ menu/
│       ├── middleware/      # 统一响应、鉴权、操作日志
│       ├── online/ token/ user/
├── utility/                 # 一次性运维脚本(建库、迁移、清理日志)
├── manifest/
│   ├── config/              # 配置(config.yaml 已忽略, 提供 config.example.yaml)
│   ├── sql/create.sql       # 建库建表脚本
│   ├── docker/              # GoFrame 模板 Dockerfile、compose 配置、Nginx 示例
│   └── deploy/kustomize/    # Kubernetes 清单
├── resource/                # 静态资源(i18n、public)
├── vue-vben-admin/          # 前端 monorepo(Vben Admin v5 + Element Plus)
├── Dockerfile               # 后端多阶段构建镜像
├── docker-compose.yml       # MySQL + Redis + 后端 一键启动
└── main.go
```

## 四、环境要求

| 组件 | 版本要求 |
| --- | --- |
| Go | **1.26+**（`go.mod` 中 `go 1.26.0`，实测 go1.27.3 可编译） |
| MySQL | 5.7+ / 8.0，字符集 `utf8mb4` |
| Redis | 6.0+ |
| Node.js | `^22.18.0` 或 `^24.12.0`（前端仓库 `engines`） |
| pnpm | ≥ 11（仓库声明 `pnpm@12.16.1`） |
| GoFrame CLI（可选） | v2.10.x，用于 `gf gen` / `gf build` |

---

## 五、快速开始（本地开发）

### 1. 克隆项目

```bash
git clone https://github.com/2213473846/demo.git
cd demo
```

### 2. 准备 MySQL 与 Redis

```bash
# Docker 快速起本地依赖
docker run -d --name vben-mysql -p 3306:3306 -e MYSQL_ROOT_PASSWORD=123456 mysql:8.0
docker run -d --name vben-redis -p 6379:6379 redis:7-alpine
```

### 3. 初始化数据库（二选一）

**方式 A：直接执行 SQL**

```bash
# Linux / macOS
mysql -uroot -p123456 < manifest/sql/create.sql

# Windows PowerShell
Get-Content manifest/sql/create.sql | mysql -uroot -p123456
```

**方式 B：使用内置脚本（会建库、建表并写入种子账号）**

```bash
go run ./utility/initdb
```

> `utility/initdb` 内的数据库连接信息是写死的（`root:123456@tcp(127.0.0.1:3306)`），如与你的环境不同，先修改该文件。
> 种子账号由服务启动时的 `internal/service/boot` 幂等创建，即使只建了空表，首次启动也会自动补齐 `admin/agent01/user01`。

### 4. 准备后端配置

```bash
# 复制示例配置后按需修改数据库、Redis 地址
cp manifest/config/config.example.yaml manifest/config/config.yaml

# Windows PowerShell
Copy-Item manifest/config/config.example.yaml manifest/config/config.yaml
```

默认配置（`manifest/config/config.yaml`）：

```yaml
server:
  address:     ":8000"
  openapiPath: "/api.json"
  swaggerPath: "/swagger"

database:
  default:
    link: "mysql:root:123456@tcp(127.0.0.1:3306)/vben_admin"

redis:
  default:
    address: "127.0.0.1:6379"
    db: 0
```

> `manifest/config/config.yaml` 已被 `.gitignore` 忽略，请勿把生产环境账号密码提交到仓库。

### 5. 启动后端

```bash
go run main.go
# 或使用 GoFrame CLI 构建后运行
gf run main.go
```

启动成功后：

- 接口服务：<http://127.0.0.1:8000>
- Swagger UI：<http://127.0.0.1:8000/swagger>
- OpenAPI JSON：<http://127.0.0.1:8000/api.json>

启动时会打印初始化日志并自动创建种子账号与内置菜单（幂等，重复启动不重复插入）。

### 6. 启动前端

```bash
cd vue-vben-admin
pnpm install          # 首次安装依赖(项目已使用 pnpm, 不要用 npm/yarn)
pnpm dev:ele          # 等价于 pnpm -F @vben/web-ele run dev
```

浏览器访问 <http://127.0.0.1:5777>。开发环境已通过 Vite 代理把 `/api` 转发到 `http://127.0.0.1:8000` 并去掉 `/api` 前缀（见 `vue-vben-admin/apps/web-ele/vite.config.ts`）。

### 7. 默认账号

| 用户名 | 密码 | 角色 | 数据权限 |
| --- | --- | --- | --- |
| `admin` | `123456` | `super_admin` | 全部用户、全部菜单 |
| `agent01` | `123456` | `admin` | 仅自己的子孙用户 |
| `user01` | `123456` | `user` | 无用户管理权限 |

> 生产环境请先登录 `admin` 修改密码或直接删除种子账号。

---

## 六、接口说明

### 统一响应格式

```json
{ "code": 0, "data": {}, "error": null, "message": "ok" }
```

- `code = 0`：成功
- `code = -1`：业务错误（HTTP 200），错误信息在 `message`
- HTTP 401：未登录 / token 过期（触发前端刷新 token）
- HTTP 403：无权限

### 认证方式

登录成功后请求头携带 `Authorization: Bearer <accessToken>`；refreshToken 由 HttpOnly Cookie `jwt` 携带，前端 401 时调用 `/auth/refresh` 换取新 token。

### 接口列表

| 方法 | 路径 | 说明 | 需登录 |
| --- | --- | --- | --- |
| POST | `/auth/login` | 登录，返回 accessToken | 否 |
| POST | `/auth/refresh` | 刷新 accessToken（读 Cookie） | 否 |
| POST | `/auth/logout` | 登出，作废 token 并清 Cookie | 否 |
| GET | `/auth/codes` | 当前用户按钮权限码 | 是 |
| GET | `/user/info` | 当前登录用户信息 | 是 |
| GET | `/user/list` | 用户列表（含数据权限过滤） | 是 |
| POST | `/user/create` | 新增用户/管理员 | 是 |
| PUT | `/user/update` | 修改用户 | 是 |
| DELETE | `/user/delete` | 软删除用户 | 是 |
| GET | `/user/recycle` | 回收站列表 | 是 |
| PUT | `/user/restore` | 恢复用户 | 是 |
| DELETE | `/user/purge` | 彻底删除用户 | 是 |
| GET | `/menu/all` | 当前用户可见路由树（后端下发） | 是 |
| GET | `/menu/list` | 菜单树（管理用） | 是 |
| POST | `/menu/create` | 新增菜单/路由 | 是 |
| PUT | `/menu/update` | 修改菜单 | 是 |
| DELETE | `/menu/delete` | 删除菜单 | 是 |
| GET | `/menu/grant` | 查询用户菜单授权 | 是 |
| PUT | `/menu/grant` | 保存用户菜单授权 | 是 |
| GET | `/loginlog/list` | 登录日志列表 | 是 |
| GET | `/operationlog/list` | 操作日志列表 | 是 |
| GET | `/online/list` | 在线用户列表 | 是 |
| POST | `/online/kick` | 强制下线指定会话 | 是 |

登录示例：

```bash
curl -X POST http://127.0.0.1:8000/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"123456"}'
```

---

## 七、新增页面 / 菜单约定

1. 在 `vue-vben-admin/apps/web-ele/src/views/` 下创建页面组件（如 `views/report/index.vue`）。
2. 登录后进入「系统管理 → 路由管理」，新增菜单：
   - `component` 填页面路径 `/report/index`（目录节点填 `BasicLayout`）
   - `path` 顶级以 `/` 开头，子级为相对路径
   - `name` 为全局唯一的路由名
3. 保存后退出重新登录（或刷新）即可看到后端下发的路由。

> 项目已切换为 Vben 的后端路由模式（`preferences.ts` 中 `accessMode: 'backend'`），前端不再维护静态业务路由。

---

## 八、常用命令

### 后端

```bash
go run main.go                 # 本地运行
go build -o main .             # 编译当前平台二进制
go vet ./...                   # 静态检查

gf gen dao                     # 依据数据库生成 dao/do/entity
gf gen ctrl                    # 依据 api/ 定义生成 controller
gf gen service                 # 生成 service 接口
gf build -ew                   # 交叉编译(配置见 hack/config.yaml, 输出到 temp/)
```

### 运维脚本（一次性，按需执行）

| 脚本 | 作用 |
| --- | --- |
| `go run ./utility/initdb` | 创建数据库、全部表并写入种子账号 |
| `go run ./utility/migratemenu` | 创建菜单表与用户菜单授权表 |
| `go run ./utility/migratelog` | 创建登录日志表与操作日志表 |
| `go run ./utility/clearlog` | 清空日志测试数据 |

### 前端

```bash
cd vue-vben-admin
pnpm install          # 安装依赖
pnpm dev:ele          # 开发模式(默认 5777)
pnpm build:ele        # 生产构建, 产物在 apps/web-ele/dist
pnpm preview          # 预览构建产物
pnpm lint             # 代码检查
```

---

## 九、部署

### 方式一：后端二进制 + Nginx（推荐，最轻量）

**1）构建前端**

```bash
cd vue-vben-admin
pnpm install
pnpm build:ele
# 产物: vue-vben-admin/apps/web-ele/dist
```

生产接口地址在 `vue-vben-admin/apps/web-ele/.env.production` 中配置，默认为同域 `/api`（由 Nginx 反向代理到后端）：

```dotenv
VITE_GLOB_API_URL=/api
```

若前后端分域部署，则改成后端完整地址（例如 `https://api.example.com`），并在后端放开 CORS（默认已启用 `ghttp.MiddlewareCORS`）。

**2）构建后端**

```bash
# Linux 服务器上直接编译
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o main .

# 或在本地交叉编译 linux/amd64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o main .
```

**3）部署目录**

```text
/opt/vben-admin/
├── main                         # 后端二进制
├── manifest/config/config.yaml  # 生产配置(自行创建, 不入库)
└── resource/                    # 静态资源, 需与二进制同目录
```

```bash
cd /opt/vben-admin && ./main
```

> GoFrame 从工作目录的 `manifest/config/config.yaml` 读取配置，务必先 `cd` 到该目录再启动。

**4）systemd 托管**

`/etc/systemd/system/vben-admin.service`：

```ini
[Unit]
Description=vben-admin backend (GoFrame)
After=network.target mysql.service redis.service

[Service]
Type=simple
WorkingDirectory=/opt/vben-admin
ExecStart=/opt/vben-admin/main
Restart=always
RestartSec=5
Environment=TZ=Asia/Shanghai
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now vben-admin
sudo systemctl status vben-admin
journalctl -u vben-admin -f
```

**5）Nginx（静态前端 + 反代接口）**

使用 `manifest/docker/nginx.conf.example`，核心两段：

```nginx
root /usr/share/nginx/html;           # 指向 dist 目录

location / {
    try_files $uri $uri/ /index.html; # hash 路由兜底
}

location /api/ {
    proxy_pass http://127.0.0.1:8000/; # 去掉 /api 前缀转发
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
}
```

```bash
sudo nginx -t && sudo systemctl reload nginx
```

部署后访问 `http://你的域名/`，接口经 `http://你的域名/api/...` 访问后端。

**6）上线检查清单**

- [ ] 修改 `config.yaml` 中的数据库账号密码，禁用 `root`
- [ ] 修改 MySQL `root` 密码与 Redis 访问控制（生产建议加 `requirepass` 并只内网暴露端口）
- [ ] 删除或修改种子账号，或为 `admin` 设置强密码
- [ ] `logger.stdout` 视情况关闭，日志目录持久化
- [ ] 生产环境关闭 Swagger（移除 `swaggerPath` 或不对外暴露 `/swagger`、`/api.json`）
- [ ] Nginx 开启 HTTPS（Vben 登录、Cookie 建议在 HTTPS 下使用）
- [ ] 数据库定期备份

### 方式二：Docker Compose（MySQL + Redis + 后端）

仓库已提供 `Dockerfile`（多阶段构建）与 `docker-compose.yml`（含 MySQL、Redis、后端）。

```bash
# 1. 启动(首次会自动执行 manifest/sql/create.sql 建库建表)
docker compose up -d --build

# 2. 查看日志/健康状态
docker compose ps
docker compose logs -f backend

# 3. 后端接口
curl http://127.0.0.1:8000/auth/login -X POST \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"123456"}'

# 4. 停止(保留数据卷)
docker compose down
# 连数据一起清理
docker compose down -v
```

说明：

- 镜像内使用 `manifest/docker/config.compose.yaml`（数据库/Redis 使用容器服务名 `mysql`、`redis`）。
- 数据持久化在 `mysql_data`、`redis_data`、`backend_logs` 卷。
- 生产环境请修改 `docker-compose.yml` 中的 MySQL 密码，并同步修改 `manifest/docker/config.compose.yaml` 的 `link`，或挂载自己的 `config.yaml`：

```yaml
    volumes:
      - ./manifest/config/config.yaml:/app/manifest/config/config.yaml:ro
```

- 前端不在 compose 内：按「方式一」构建 `dist` 后，用 Nginx 容器或宿主机 Nginx 托管，并把 `/api/` 反代到 `http://127.0.0.1:8000/`。

```bash
# 仅构建后端镜像
docker build -t vben-admin-backend:latest .
```

### 方式三：Kubernetes（现有清单）

`manifest/deploy/kustomize` 中提供了 base 与 develop overlay 示例：

```bash
# 预览渲染结果
kubectl kustomize manifest/deploy/kustomize/overlays/develop

# 应用(注意先替换镜像名与配置)
kubectl apply -k manifest/deploy/kustomize/overlays/develop
```

使用前需要：构建并推送你的后端镜像，修改 `deployment.yaml` 中的 `image` 与 `service.yaml` 端口，并通过 ConfigMap 提供 `config.yaml`（overlay 中已有示例）。

---

## 十、配置说明（`manifest/config/config.yaml`）

| 配置项 | 说明 |
| --- | --- |
| `server.address` | 监听地址，默认 `:8000` |
| `server.openapiPath` | OpenAPI 描述文件路径，默认 `/api.json` |
| `server.swaggerPath` | Swagger UI 路径，默认 `/swagger` |
| `logger.level` / `logger.stdout` | 日志级别与是否输出到控制台 |
| `database.default.link` | MySQL 连接串：`mysql:user:pass@tcp(host:port)/dbname` |
| `redis.default.address` / `db` / `pass` | Redis 连接信息 |

其余可调项参考 GoFrame 官方文档：<https://goframe.org/docs/core/gcfg-config-file>。

## 十一、常见问题（FAQ）

**1. 启动报「数据库不可用」**

MySQL 未启动或配置不正确。请确认 `manifest/config/config.yaml` 的连接串，并先执行 `go run ./utility/initdb` 或导入 `manifest/sql/create.sql`。

**2. 登录成功但请求接口返回 401 / 刷新后又失效**

token 存在 Redis 中。请确认 Redis 已启动、`redis.default.address` 正确；Redis 不可用时登录会失败，accessToken 无法校验。

**3. 登录后菜单是空白的**

本项目路由由后端下发（`accessMode: 'backend'`）。请检查：`sys_menu` 是否有数据、当前用户是否有菜单授权（「路由管理 → 菜单授权」）、`/menu/all` 是否正常返回。

**4. 前端本地开发请求 404 或跨域**

开发环境依赖 Vite 代理（`/api` → `127.0.0.1:8000`，并去掉 `/api` 前缀）。请确认后端已在 8000 端口启动，且前端使用 `pnpm dev:ele` 启动。

**5. 生产构建后仍请求 mock 地址**

检查 `vue-vben-admin/apps/web-ele/.env.production` 的 `VITE_GLOB_API_URL`，本项目默认改为 `/api`；如自定义需重新执行 `pnpm build:ele`。

**6. Windows 下 `mysql < create.sql` 报错**

PowerShell 不支持 `<` 输入重定向，请使用 `Get-Content manifest/sql/create.sql | mysql -uroot -p123456`，或在 `cmd` 下执行。

**7. 前端安装依赖失败**

必须使用 pnpm（仓库 `preinstall` 会强制校验），Node 版本需满足 `^22.18.0 || ^24.12.0`，国内网络可配置 `npm_config_registry=https://registry.npmmirror.com`。

## 十二、致谢与许可

- 前端基于 [Vue Vben Admin](https://github.com/vbenjs/vue-vben-admin)（MIT License）
- 后端基于 [GoFrame v2](https://github.com/gogf/gf)

本项目为个人学习/演示项目，请遵循所依赖开源项目的许可协议使用。
