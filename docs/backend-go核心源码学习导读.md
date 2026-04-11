# backend-go 核心源码学习导读

本文面向 Go 初学者，基于 `backend-go/` 的真实源码，挑出一组最典型、最核心的文件做“文件内函数 / 代码块级”解释。

这份文档的目标不是替代源码，而是帮助你建立 3 个基本感觉：

1. 服务是怎样启动起来的。
2. 一个 HTTP 请求是怎样一路走到业务层和数据层的。
3. Go 项目里常见的 `struct`、`interface`、`context`、`error`、事务和中间件分别落在什么位置。

## 1. 推荐阅读顺序

如果你第一次读这个项目，建议按下面顺序看：

1. `cmd/api/main.go`
2. `internal/config/config.go`
3. `internal/app/server.go`
4. `internal/bootstrap/container.go`
5. `internal/transport/http/router/router.go`
6. `internal/transport/http/middleware/auth.go`
7. `internal/transport/http/handlers/auth.go`
8. `internal/transport/http/handlers/documents.go`
9. `internal/transport/http/handlers/versions.go`
10. `internal/service/auth_service.go`
11. `internal/service/token_service.go`
12. `internal/service/document_service.go`
13. `internal/service/version_service.go`
14. `internal/service/flow_service.go`
15. `internal/service/assistant_service.go`
16. `internal/repository/contracts.go`
17. `internal/repository/postgres/document_repository.go`
18. `internal/repository/postgres/version_workflow.go`
19. `internal/storage/contracts.go`

前 5 个文件帮助你建立“系统怎么跑起来”的主线；后面的文件帮助你理解“请求怎么流动、业务怎么编排、数据怎么落地”。

## 2. 先记住一条总链路

在这个项目里，一条典型请求链路大致是：

1. 浏览器把请求发到某个 URL。
2. `router` 把 URL 交给某个 handler。
3. handler 负责解析参数、读取当前用户、调用 service。
4. service 负责业务校验和业务编排。
5. repository 负责读写 PostgreSQL 或内存实现。
6. storage 负责文件上传下载。
7. response 工具负责把结果包成统一 JSON。

带着这条链路去读下面每个文件，会更容易理解每一层为什么存在。

## 3. `cmd/api/main.go`

这个文件是程序入口，只有一个核心函数：`main()`。

### `main()`

作用：启动整个 Go API 进程。

关键代码块：

1. `cfg := config.Load()`
   - 先加载配置。
   - 这是很多 Go 服务的固定起点。
   - 初学者要建立一个意识：程序一启动，先把配置归一成结构体，后面所有模块都围绕这份配置工作。

2. `server, err := app.NewServer(cfg)`
   - 这里把“创建 HTTP 服务”这件事交给 `app` 层。
   - `main` 自己不关心路由怎么装、数据库怎么连，只负责组装和启动。

3. `log.Printf("starting %s on %s", ...)`
   - 启动前打印日志，方便排查启动失败时卡在哪个阶段。

4. `server.ListenAndServe()`
   - 真正开始监听端口。
   - 返回 error 时直接 `log.Fatal`，说明这是一个“前台常驻服务”。

学习点：

- `main` 要尽量薄，不写业务逻辑。
- 启动顺序通常是“读配置 -> 建依赖 -> 起服务”。
- Go 很常见的错误处理模式就是“拿到 error 立刻判断”。

## 4. `internal/config/config.go`

这个文件负责把环境变量变成程序可用的配置对象。

### `type Config struct`

作用：集中保存运行所需的配置。

你可以把它理解成“系统的启动参数总表”。它里面最值得关注的是：

- `HTTPAddr`：服务监听地址。
- `DatabaseURL`：PostgreSQL 连接串。
- `DataBackend`：当前用 `memory` 还是 `postgres` 作为数据后端。
- `StorageBackend`：当前用 `memory` 还是 `synology` 作为文件存储后端。
- `JWTSecret`：JWT 签名密钥。
- 一组 `Synology*` 字段：群晖 File Station 适配器的配置。

学习点：

- Go 项目里常把配置集中在一个 struct 中，而不是到处 `os.Getenv`。
- 后续每一层都依赖 `Config`，所以配置层通常是系统最早初始化的模块之一。

### `Load()`

作用：从环境变量读取配置并返回 `Config`。

关键代码块：

1. `strconv.Atoi(getEnv("SYNOLOGY_PORT", "5000"))`
   - 环境变量本质上都是字符串。
   - 这个代码块演示了如何把字符串转成 `int`。

2. 构造 `cfg := Config{ ... }`
   - 这一大段是“把分散的环境变量收口到统一配置对象里”。
   - 对初学者来说，这是很典型的“初始化数据结构”写法。

3. `if cfg.AppEnv == "production" { ... }`
   - 生产环境下做安全校验。
   - 例如不能继续使用默认 JWT 密钥和默认 Worker token。
   - 这说明配置层不只是“读取”，还会做“兜底校验”。

学习点：

- `Load` 通常兼顾“读取 + 默认值 + 基础校验”三件事。
- 启动时尽早失败，比服务启动后半路报错更容易排查。

### `getEnv(key, fallback)`

作用：封装“有环境变量就用环境变量，没有就用默认值”的逻辑。

学习点：

- 这是一个非常常见的小工具函数。
- 初学者可以体会：很多 Go 项目喜欢先写几个很小的 helper，减少重复代码。

## 5. `internal/app/server.go`

这个文件把配置和容器拼成一个真正可运行的 `http.Server`。

### `NewServer(cfg)`

作用：创建 HTTP server。

关键代码块：

1. `container, err := bootstrap.BuildContainer(cfg)`
   - 先把所有依赖装好。
   - 所有 service、repository、storage 都在这里之前被准备好。

2. `handler := httprouter.New(cfg, container)`
   - 基于配置和依赖容器生成路由。
   - 可以理解为“把 URL 映射规则拼成一个总入口”。

3. 返回 `&http.Server{ ... }`
   - 用 Go 标准库的 `http.Server` 封装监听地址、handler 和超时。
   - `ReadHeaderTimeout` 是一个基础的防御性配置，避免请求头无限拖延。

学习点：

- `app` 层通常是运行壳层，不写具体业务。
- 它负责把更底层的组件装成一个可以对外提供服务的对象。

## 6. `internal/bootstrap/container.go`

这是整个项目最适合学习“依赖注入”和“按配置切换实现”的文件之一。

### `type Container struct`

作用：把系统里会被复用的依赖集中放在一起。

这个结构体里装的是：

- `DB`
- `QueueConsumer`
- 各类 Service
- `TokenService`
- `AuditService`

学习点：

- 容器不是为了“炫技”，而是为了集中管理依赖。
- handler 和 router 只需要拿 container，就能拿到自己需要的 service。

### `BuildContainer(cfg)`

作用：根据配置选择不同实现，并把它们组装成完整容器。

这个函数是整份源码里最值得慢慢读的函数之一。

关键代码块：

1. 先创建公共依赖
   - `publisher := memqueue.NewPublisher()`
   - `storageProvider := buildStorageProvider(cfg)`
   - `tokenService := service.NewTokenService(cfg.JWTSecret)`
   - `auditService := service.NewAuditService()`
   - 这部分说明：有些依赖不区分 `memory` / `postgres`，可以先统一创建。

2. `switch cfg.DataBackend`
   - 这是最核心的装配分支。
   - 如果是 `postgres`，就创建真实数据库实现。
   - 否则使用内存实现。
   - 这正是 interface 的价值：上层 service 只依赖接口，不依赖某一种具体实现。

3. `postgres` 分支
   - `db.OpenPostgres(...)` 打开数据库。
   - `findMigrationsDir()` 查迁移目录。
   - `db.RunMigrations(...)` 尝试执行迁移。
   - 然后创建一整套 Postgres repository，再注入到 service 中。
   - 这说明：启动服务时不仅要连上数据库，还要尽量确保 schema 已经到位。

4. `memory` 分支
   - 创建 memory repository 和 memory workflow。
   - 适合开发和测试。
   - 这也是为什么仓储层要用接口抽象。

5. 返回 `Container{ ... }`
   - 整个函数最后返回的是“已经装配好的应用依赖树”。

学习点：

- 这个函数很好地体现了 Go 项目的组合式设计。
- service 构造函数一般很简单，重点是“依赖怎么传进去”。
- 真正的“架构味道”往往体现在装配方式，而不只是在业务函数里。

### `findMigrationsDir()`

作用：定位迁移文件目录。

关键代码块：

1. 先尝试工作目录下的候选路径。
2. 再尝试相对当前源码文件的位置去找。
3. 用 `filepath.Glob(...*.sql)` 判断目录里是否真的有 SQL 文件。

学习点：

- 这类函数是很典型的“运行环境兼容性辅助逻辑”。
- 它不复杂，但非常工程化，解决的是“不同启动方式下路径不一致”的问题。

### `buildStorageProvider(cfg)`

作用：根据配置返回 `storage.Provider` 的具体实现。

关键代码块：

1. `case "synology"`
   - 校验必要配置，例如 `SYNOLOGY_HOST`。
   - 然后构造群晖适配器。

2. `default`
   - 回退到内存存储。
   - 对本地开发非常友好。

学习点：

- 这是标准的“工厂函数”思路。
- 上层只认 `storage.Provider` 接口，不关心底层到底是群晖还是内存。

## 7. `internal/transport/http/router/router.go`

这个文件回答了两个问题：

1. 哪些 URL 存在。
2. 每个 URL 由哪个 handler 处理。

### `New(cfg, container)`

作用：创建整个 HTTP 路由树。

关键代码块：

1. `mux := http.NewServeMux()`
   - 使用标准库路由器。
   - 说明这个项目不依赖 Gin/Echo 这类框架，而是基于标准库自己组织结构。

2. 创建一批 handler
   - 例如 `NewAuthHandler`、`NewDocumentHandler`、`NewVersionHandler`。
   - 这里能清楚看出 router 依赖的是 service，而不是 repository。

3. `authMw := middleware.Auth(container.TokenService)`
   - 把鉴权中间件先准备好。

4. `protect := func(h http.HandlerFunc) http.Handler { ... }`
   - 这是一个小包装器，方便给所有受保护接口统一加 JWT 校验。

5. Public routes
   - 例如 `/healthz`、`/auth/login`、Worker 内部接口。
   - 这些接口不依赖普通用户 JWT。

6. Protected routes
   - 如 `/auth/me`、`/documents`、`/versions`、`/flows`、`/handovers`。
   - 它们都统一包上 `protect(...)`。

7. 最后 `middleware.Chain(...)`
   - 统一挂上 CORS、RequestID、JSONContentType、AccessLog。
   - 这是横切逻辑集中收口的地方。

学习点：

- router 层只处理“URL 到 handler 的映射”，不写业务规则。
- 中间件顺序有意义，尤其是日志、请求 ID、鉴权这几类中间件。

## 8. `internal/transport/http/middleware/auth.go`

这是理解 Go 中间件最好的示例之一。

### `Auth(tokenService)`

作用：校验 Bearer Token，并把 claims 注入请求上下文。

关键代码块：

1. 读取 `Authorization` 请求头。
2. 检查是否以 `Bearer ` 开头。
3. 调用 `tokenService.Parse(token)` 验证签名和过期时间。
4. `context.WithValue(...)` 把 claims 塞进上下文。
5. `next.ServeHTTP(...)` 把处理权交给下游 handler。

学习点：

- 中间件的本质是“在进入真正 handler 前后插入公共逻辑”。
- 当前用户信息不适合放全局变量，因为 HTTP 请求是并发的。
- `context` 是每个请求自己的数据容器。

### `ClaimsFromContext(ctx)`

作用：从上下文中取出鉴权后的 claims。

学习点：

- 这类 helper 可以避免 handler/service 自己重复写类型断言。

### `UserIDFromContext(ctx)`

作用：提取当前用户 ID。

注意点：

- 如果没拿到 claims，会回退成系统用户 ID。
- 这意味着有些内部链路也允许系统身份执行动作。

## 9. `internal/transport/http/middleware/request_id.go`

这个文件体现了“请求链路追踪”的最小实现。

### `RequestID(next)`

作用：给每个请求分配请求 ID，并写入响应头与上下文。

关键代码块：

1. 优先读取客户端传来的 `X-Request-Id`。
2. 没有就调用 `newRequestID()` 自己生成。
3. 把 request ID 写回响应头，方便前后端对齐日志。
4. 调用 `shared.WithRequestID(...)` 把它放进上下文。

学习点：

- 很多中间件不是做业务，而是做“可观测性”。
- request ID 能把 access log、审计、异步任务串起来。

### `newRequestID()`

作用：生成一个足够唯一的请求 ID。

关键代码块：

- `atomic.AddUint64` 保证并发下序列号安全递增。
- `time.Now().UnixNano()` 结合序列号构造字符串。

学习点：

- 这里能看到 Go 标准库对并发安全的基础支持。
- `atomic` 是初学者值得认识的包，但不必一开始就深挖。

## 10. `internal/transport/http/handlers/auth.go`

这个文件展示了最标准的“解析参数 -> 调 service -> 写响应”模式。

### `type AuthHandler`

作用：把认证相关依赖组织到一个 handler 结构体里。

字段含义：

- `authService`：真正处理登录逻辑。
- `tokenService`：处理 token 解析。

### `NewAuthHandler(...)`

作用：构造 `AuthHandler`。

学习点：

- Go 里很常见这种简单构造函数。
- 它不是必须语法，但能统一依赖注入方式。

### `Login(w, r)`

作用：处理登录请求。

关键代码块：

1. `json.NewDecoder(r.Body).Decode(&payload)`
   - 解析 JSON 请求体。

2. 校验 `Username` 和 `Password` 是否为空。
   - 这是 HTTP 层最基础的输入校验。

3. `h.authService.Login(...)`
   - 真正的认证逻辑交给 service。

4. 错误映射
   - `ErrUnauthorized` 映射成 401。
   - 其他错误映射成 500。

5. `response.WriteData(...)`
   - 成功时返回 token 和 user 信息。

学习点：

- handler 不做密码比对，不查数据库。
- handler 的职责是协议层适配，而不是业务编排。

### `Me(w, r)`

作用：获取当前登录用户信息。

关键代码块：

1. 先从请求头里取 Bearer Token。
2. 再用 `tokenService.Parse` 解析 claims。
3. 把 claims 重组为前端需要的返回结构。

说明：

- 这里没有再去查数据库，而是直接信任 JWT 中的 claims。
- 这是很多系统里常见的性能换取简洁的做法。

### `Logout(w, r)`

作用：返回一个成功标记。

说明：

- 当前实现是无状态 JWT，所以退出登录并不需要服务端销毁 session。
- 这里更像一个前端协作接口。

## 11. `internal/transport/http/handlers/documents.go`

这个文件是文档主业务的 HTTP 入口，适合反复读。

### `type DocumentHandler` 与 `NewDocumentHandler`

作用：保存 `DocumentService`，并提供构造函数。

### `Create(w, r)`

作用：创建文档并上传首个版本。

关键代码块：

1. `r.ParseMultipartForm(shared.MaxUploadSize)`
   - 说明这是一个 `multipart/form-data` 接口，而不是普通 JSON 接口。

2. `r.FormFile("file")`
   - 取上传文件流和文件头。
   - `defer file.Close()` 是非常典型的 Go 资源释放写法。

3. `shared.ValidateFileName(...)`
   - 用共享工具校验扩展名是否合法。

4. `middleware.UserIDFromContext(...)`
   - 从鉴权上下文里拿当前用户。

5. 组装 `command.DocumentCreateInput`
   - 这一步把 HTTP 表单转换为业务层输入模型。

6. `h.service.CreateWithFirstVersion(...)`
   - 交给 service 做“文档记录 + 文件上传 + 首版本创建”的组合操作。

7. 错误映射与成功响应。

学习点：

- handler 层经常负责把 `http.Request` 转成 `command`。
- 文件上传接口通常比普通 JSON 接口多一个“文件流处理”的步骤。

### `List(w, r)`

作用：按过滤条件分页列出文档。

关键代码块：

1. 从 query string 读取筛选条件。
2. 组装 `query.DocumentListFilter`。
3. `parseIntOrDefault` 给分页参数兜底。
4. 调用 `ListDocuments`。
5. 用 `response.WriteWithMeta` 同时返回 `data` 和 `meta`。

学习点：

- `query` 模型通常用来表示查询条件和查询结果。
- 分页接口常见模式是“数据 + 元信息”。

### `Get(w, r)`

作用：按路径参数获取文档详情。

关键代码块：

1. `r.PathValue("documentID")` 取路径参数。
2. `h.service.GetDocument(...)` 获取业务数据。
3. 区分 `ErrNotFound` 和其他错误。

### `Update(w, r)`

作用：更新标题、描述、目录。

关键代码块：

1. 解析 JSON body。
2. 从 context 取 `actorID`。
3. 组装 `command.DocumentUpdateInput`。
4. 调 service。

学习点：

- 写接口常见模式是“body + path + actorID”三类输入合并。

### `Delete(w, r)`

作用：软删除文档。

关键代码块：

1. 解析删除原因。
2. 组装 `command.DocumentDeleteInput`。
3. 调 service 删除。
4. 返回 `is_deleted: true`。

### `Restore(w, r)`

作用：恢复已软删除文档。

关键代码块：

1. 读取路径参数。
2. 读取当前用户。
3. 调 service 恢复。
4. 返回 `is_deleted: false`。

### `parseIntOrDefault(raw, fallback)`

作用：把 query string 中的字符串分页参数转成 int，并兜底。

学习点：

- 这类小函数体现 Go 项目里“把重复的小逻辑提炼出来”的习惯。

## 12. `internal/transport/http/handlers/versions.go`

这个文件目前只有一个最核心的方法，说明版本上传入口是单独收口的。

### `type VersionHandler` 与 `NewVersionHandler`

作用：保存 `VersionService` 并提供构造函数。

### `Upload(w, r)`

作用：为指定文档上传一个新版本。

关键代码块：

1. 解析 multipart 表单。
2. 读取文件流。
3. 校验扩展名。
4. 从路径里拿 `documentID`。
5. 从上下文里拿当前用户 ID。
6. 调 `UploadAndCreateVersion`。
7. 把校验错误映射成 400。

和 `DocumentHandler.Create` 的区别：

- 这里不是“创建文档”，而是“已有文档上新增版本”。
- 所以只需要 `documentID`，不再需要团队空间、项目等元信息。

## 13. `internal/transport/http/response/response.go`

这是 HTTP 响应统一格式的工具文件。

### `type Envelope`

作用：统一成功和失败返回格式。

字段含义：

- `Data`：成功时的主体数据。
- `Meta`：分页等附加信息。
- `Code`：错误码。
- `Error`：错误消息，JSON 字段名是 `message`。

### `WriteData(...)`

作用：返回只有 `data` 的成功响应。

### `WriteWithMeta(...)`

作用：返回 `data + meta` 的成功响应。

### `WriteError(...)`

作用：返回统一错误结构。

### `writeJSON(...)`

作用：所有对外写 JSON 的底层实现。

关键代码块：

1. 先设置 `Content-Type`。
2. 再写状态码。
3. 用 `json.NewEncoder(w).Encode(payload)` 输出。
4. 如果编码失败，再回退到 `http.Error`。

学习点：

- 统一响应格式是后端项目的基础卫生工作。
- 这样 handler 不必每次手写 JSON 序列化。

## 14. `internal/shared/upload.go`

这个文件是上传相关的共享规则。

### `AllowedFileExtensions`

作用：上传白名单。

学习点：

- 这种规则适合放在 shared 层，而不是散落在每个 handler 里。

### `MaxUploadSize`

作用：限制最大上传体积，这里是 32 MB。

### `ValidateFileName(fileName)`

作用：根据文件扩展名判断是否允许上传。

关键代码块：

1. `filepath.Ext(fileName)` 提取扩展名。
2. `strings.ToLower(...)` 统一大小写。
3. 到白名单 map 里查是否允许。

学习点：

- Go 里 map 非常适合做白名单 / 黑名单查询。

## 15. `internal/service/auth_service.go`

这个文件体现了“service 做业务编排，不做 HTTP 适配”的思想。

### `type AuthService`

字段含义：

- `userRepo`：负责查用户。
- `tokenSvc`：负责签发 JWT。

### `NewAuthService(...)`

作用：构造认证服务。

### `Login(ctx, username, password)`

作用：校验用户名密码，并返回签名后的 JWT 和 claims。

关键代码块：

1. `s.userRepo.FindUserByUsername(...)`
   - 通过仓储查用户。
   - 如果底层返回 `ErrNotFound`，这里把它转换成 `ErrUnauthorized`。
   - 这是很重要的“错误语义转换”：对外不暴露“用户名不存在”这种过细信息。

2. `bcrypt.CompareHashAndPassword(...)`
   - 校验明文密码和数据库里的密码哈希是否匹配。
   - 这一步是认证核心。

3. 构造 `auth.Claims`。
   - 把用户信息整理成 token 需要的 claims。

4. `s.tokenSvc.Generate(claims)`
   - 生成 JWT。

学习点：

- service 层负责“业务含义”而不是“HTTP 状态码”。
- 错误从仓储层向上冒泡时，经常要做语义转换。

## 16. `internal/service/token_service.go`

这个文件非常适合初学者理解“一个可独立测试的纯逻辑服务”。

### `type TokenService` 与 `NewTokenService`

作用：保存 JWT 签名密钥，并提供构造函数。

### `Generate(claims)`

作用：生成一个 2 小时有效的 HS256 JWT。

关键代码块：

1. 构造 payload map。
2. `json.Marshal(payload)` 转成 JSON。
3. Base64URL 编码。
4. 拼出 `header.payload`。
5. `s.sign(unsigned)` 生成签名。
6. 返回 `header.payload.signature`。

学习点：

- 这个实现没有依赖第三方 JWT 库，而是自己按标准拼装。
- 对初学者来说，能很好地看到 JWT 的结构本质。

### `Parse(token)`

作用：验证 token 格式、签名和过期时间，然后还原 claims。

关键代码块：

1. `strings.Split(token, ".")` 切出 3 段。
2. 重算签名并与原签名比较。
3. Base64URL 解码 payload。
4. `json.Unmarshal` 成 map。
5. 检查 `exp` 是否过期。
6. 把 payload map 再转回 `auth.Claims`。

学习点：

- 这是一个很好的“解析字符串协议”的例子。
- 也能看到 Go 对二进制、JSON、字符串处理的组合方式。

### `sign(input)`

作用：用 HMAC-SHA256 对 `header.payload` 做签名。

### `stringClaim(payload, key)`

作用：从通用 map 中安全提取字符串 claim。

### `ExtractBearerToken(authHeader)`

作用：从 `Authorization: Bearer <token>` 中截出 token。

说明：

- 这个函数本身很小，但复用性很高。

## 17. `internal/service/document_service.go`

这是文档主业务编排层。

### `type DocumentService` 与 `NewDocumentService`

字段含义：

- `reader`：负责文档查询。
- `writer`：负责文档写操作。
- `storage`：负责文件上传。
- `workflow`：负责版本事务工作流。

学习点：

- 一个 service 同时依赖多个接口非常正常，因为它要编排多个下游能力。

### `ListDocuments(ctx, filter)`

作用：把查询请求直接转交给 reader。

说明：

- 这是“薄封装”函数。
- 对初学者来说，要认识到不是每个 service 函数都很复杂，有些只是统一入口。

### `GetDocument(ctx, documentID)`

作用：获取文档详情，也是一个薄封装。

### `UpdateDocument(ctx, input)`

作用：更新文档基本信息。

关键代码块：

1. 校验 `DocumentID` 必填。
2. 校验至少要有一个字段被更新。
3. 调 `writer.UpdateDocument(...)`。

学习点：

- service 层很适合放“输入合法性校验”。

### `DeleteDocument(ctx, input)`

作用：删除文档前先检查 `document_id` 是否为空，再委托 writer。

### `RestoreDocument(ctx, documentID, actorID)`

作用：恢复文档前先做基本参数校验，再委托 writer。

### `CreateWithFirstVersion(...)`

作用：创建文档并同时创建首个版本。

这是本文件最值得重点学习的函数。

关键代码块：

1. 一组必填校验
   - `team_space_id`
   - `project_id`
   - `title`
   - `current_owner_id`
   - `actor_id`
   - `fileName`
   - 这些校验保证后续链路不必再猜输入是否合法。

2. `s.writer.CreateDocument(...)`
   - 先创建文档主记录。
   - 返回的 `docData` 里会带文档 ID。

3. 构造 `objectKey`
   - 例如 `documents/<documentID>/<fileName>`。
   - 这是文件在存储层中的路径约定。

4. `s.storage.PutObject(...)`
   - 真正上传文件内容。

5. `s.workflow.CreateUploadedVersion(...)`
   - 把版本元数据写入数据库，并推进文档当前版本和状态。
   - 这是“文件存储 + 数据库事务”串联的关键点。

6. 回填 `docData["current_owner"]` 和 `docData["current_version"]`
   - 把最终前端需要的信息补齐到返回结构中。

学习点：

- 这是一个标准的“业务编排函数”。
- service 不直接写 SQL，但它决定了动作顺序。
- 对初学者来说，重点不是记住每一行，而是理解“为什么要先文档、再上传、再版本工作流”。

## 18. `internal/service/version_service.go`

这个文件是“已有文档上传新版本”的业务层。

### `type VersionService` 与 `NewVersionService`

字段含义：

- `storage`：上传下载文件。
- `workflow`：写版本事务。
- `reader`：读版本数据。

### `UploadAndCreateVersion(...)`

作用：上传新文件并创建一个新的版本记录。

关键代码块：

1. 校验 `documentID`、`fileName`、`actorID`。
2. 构造存储路径 `documents/<documentID>/<fileName>`。
3. `s.storage.PutObject(...)` 上传文件。
4. `s.workflow.CreateUploadedVersion(...)` 写版本事务。
5. 在返回结果里补 `storage` 和 `status` 信息。

学习点：

- 这个函数和 `CreateWithFirstVersion` 很像，但少了“创建文档主记录”那一步。
- 两个函数一起对比，非常适合理解“首版本”和“追加版本”的差异。

### `List(ctx, documentID)`

作用：获取一个文档的版本列表。

### `Get(ctx, versionID)`

作用：获取单个版本详情。

### `GetFile(ctx, versionID)`

作用：先查版本元信息，再去存储层取文件内容。

关键代码块：

1. `s.reader.GetVersion(...)` 查出 `StorageObjectKey`。
2. `s.storage.GetObject(...)` 取真实文件流。
3. 返回“版本详情 + 文件对象”。

学习点：

- 这体现了“元信息存在数据库，二进制文件存在存储系统”的常见设计。

## 19. `internal/service/flow_service.go`

这个文件负责文档流转动作。

### `validFlowActions`

作用：列出允许的流转动作。

学习点：

- Go 里常用 `map[string]bool` 做枚举白名单校验。

### `type FlowService` 与 `NewFlowService`

字段含义：

- `reader`：查流转记录。
- `writer`：写流转动作。

### `ApplyAction(ctx, input)`

作用：执行一个文档流转动作。

关键代码块：

1. 校验 `DocumentID`。
2. 校验 action 是否在白名单里。
3. 如果是 `transfer`，要求 `ToUserID` 必填。
4. 如果是 `transfer`，禁止转交给自己。
5. 校验 `ActorID`。
6. 调 `writer.CreateFlowRecord(...)` 落真正动作。

学习点：

- service 层非常适合放状态机输入校验。
- “动作是否合法”往往先于“数据库怎么写”。

### `ListFlows(ctx, documentID)`

作用：列出某个文档的流转历史。

## 20. `internal/service/assistant_service.go`

这是项目里最能体现“异步任务 + 会话 + AI 请求归档”的 service。

### `type AssistantAskResult`

作用：定义 `Ask` 的返回结构。

字段里最值得注意的是：

- `RequestID`：异步任务主键。
- `ConversationID`：会话主键。
- `SourceScope`：这次问答绑定的业务范围。
- `MemorySources`：这次构建上下文时使用的记忆来源。

### `type AssistantService` 与 `NewAssistantService`

字段含义：

- `publisher`：负责把任务发布到队列。
- `repo`：负责 AI 请求、会话、建议等主账本。
- `documents`：辅助做文档范围归一化。

### `Ask(ctx, payload, actorID)`

作用：接收一次 AI 问答请求，并把它转成异步任务。

这是本文件最关键的函数。

关键代码块：

1. 提取并校验 `question`
   - 问题不能为空。

2. `clonePayload(payload)`
   - 先复制 payload，避免直接修改调用方原始数据。
   - 这是比较稳妥的 defensive coding。

3. 提取 `scope` 和 `conversation_id`
   - 判断这是新会话还是已有会话上的继续提问。

4. 如果带了 `conversation_id`
   - 先去 repo 查原会话。
   - 再把旧会话 scope 和本次 scope 合并。
   - 这是为了保证同一会话的上下文范围一致。

5. `normalizeScope(...)`
   - 把输入的业务范围标准化。
   - 这是这类 AI 系统很关键的步骤：先明确“你在问哪个项目/文档/交接单”。

6. 如无会话则创建新会话
   - `CreateConversation(...)`。

7. `buildMemorySnapshot(...)`
   - 根据会话和 scope 生成给 AI 用的上下文快照。

8. 把 `conversation_id`、`scope`、`memory`、`memory_sources` 写回 payload。
   - 为后续 Worker 处理准备完整任务载荷。

9. `s.QueueTask(...)`
   - 真正入库并发布到队列。

学习点：

- 这类函数不是同步算答案，而是“创建任务 + 保存上下文 + 返回排队状态”。
- 它很适合帮助初学者建立对异步系统的直觉。

### `QueueTask(ctx, taskType, relatedType, relatedID, payload, actorID)`

作用：把任务同时写入主账本并发布到队列。

关键代码块：

1. 构造 `task.Message`。
2. 如果 payload 为 nil，就补成空 map。
3. `s.repo.CreateAssistantRequest(...)` 先把请求落库。
4. `s.publisher.Publish(...)` 再发布给下游消费者。

学习点：

- 先落库再发布，是异步系统里很常见的稳妥顺序。
- 这样即使后续消费链路有问题，主账本也能查到请求记录。

### `ReceiveResult(ctx, result)`

作用：接收 Worker 回写结果，并更新请求状态。

### `GetRequest(ctx, requestID)`

作用：查询单个 AI 请求。

### `ListRequests(ctx, filter)`

作用：按条件列出 AI 请求。

### `CreateConversation(ctx, scopePayload, title, actorID)`

作用：创建新会话。

关键点：

- 先 `normalizeScope`，再创建会话。
- 说明会话必须绑定明确业务范围。

### `ListConversations(ctx, filter)`

作用：列出会话。

### `GetConversation(ctx, conversationID)`

作用：读取单个会话。

### `ListConversationMessages(ctx, conversationID)`

作用：列出会话消息。

关键点：

- 它会先确认会话存在，再去读消息列表。
- 这属于“先校验主对象存在性，再读从属对象”的常见模式。

### `GetLatestDocumentExtractedText(ctx, documentID)`

作用：获取文档最新抽取文本，给 AI 摘要或问答做辅助上下文。

### `ListSuggestions(ctx, filter)`

作用：列出 AI 建议。

### `ConfirmSuggestion(ctx, suggestionID, actorID, note)`

作用：确认一条 AI 建议。

### `DismissSuggestion(ctx, suggestionID, actorID, reason)`

作用：忽略一条 AI 建议。

学习点：

- 这几个函数本身偏薄封装，但它们让 `AssistantService` 成为 AI 相关业务的统一入口。

## 21. `internal/repository/contracts.go`

这个文件最适合理解 Go 里的 interface 是怎么服务于分层设计的。

### 这一层在做什么

它没有 SQL，也没有业务流程。它只定义“数据层应该提供哪些能力”。

### 代表性接口

1. `DocumentReader` / `DocumentWriter`
   - 把文档读操作和写操作分开。

2. `VersionReader` / `VersionWorkflow`
   - 读版本和“版本事务工作流”是两类不同能力。

3. `FlowReader`、`HandoverReader`、`AuditReader`、`DashboardReader`
   - 都是在定义查询边界。

4. `ActionWriter`
   - 把流转、交接等动作写链收口。

5. `AssistantRepository`
   - 一口气定义会话、请求、建议等 AI 主账本能力。

学习点：

- interface 不是为了让代码看起来高级，而是为了隔离上层和下层。
- 同一个 service 可以依赖多个小接口，而不是依赖一个巨大的万能接口。

## 22. `internal/repository/postgres/document_repository.go`

这是文档真实落库实现，适合在理解完 service 后再读。

### `type DocumentRepository` 与 `NewDocumentRepository`

作用：保存 `DBTX` 并提供构造函数。

这里的 `DBTX` 很重要，它允许同一套 repository 逻辑既能跑在 `*sql.DB` 上，也能跑在 `*sql.Tx` 上。

### `ListDocuments(ctx, filter)`

作用：分页查询文档列表。

关键代码块：

1. 先修正 `page` 和 `pageSize` 的默认值。
2. `QueryContext(...)` 执行列表 SQL。
3. `defer rows.Close()` 释放数据库游标。
4. `for rows.Next()` + `rows.Scan(...)` 把每一行组装成 `query.DocumentListItem`。
5. `rows.Err()` 检查遍历过程是否出错。
6. 再用 `QueryRowContext(...)` 执行一次 `COUNT` 查总数。

学习点：

- 这是 Go 标准 `database/sql` 代码的典型写法。
- 列表查询通常分成“两段”：查当前页数据，查总数。

### `GetDocument(ctx, documentID)`

作用：查询单个文档详情。

关键代码块：

1. `QueryRowContext(...)` 查一条记录。
2. `Scan(...)` 到 `item` 和 `owner`。
3. 如果是 `sql.ErrNoRows`，转换成 `service.ErrNotFound`。

学习点：

- repository 也会做一层错误语义转换，不只是 service 才做。

### `CreateDocument(ctx, input)`

作用：插入文档主记录。

关键代码块：

1. `id := newID()` 生成 UUID。
2. `now := time.Now().UTC()` 统一使用 UTC 时间。
3. `ExecContext(...)` 执行 `INSERT`。
4. 返回一个简化的 map 作为写操作结果。

学习点：

- 初学者要注意：并不是所有仓储函数都必须返回完整 struct。
- 有时候只返回业务层真正需要的最小信息。

### `UpdateDocument(ctx, input)`

作用：更新文档并返回更新后的关键信息。

关键代码块：

1. 用 `COALESCE` / `NULLIF` 处理可选字段。
2. `RETURNING` 直接把更新后的字段查出来。
3. 如果没有更新到任何记录，返回 `ErrNotFound`。

学习点：

- `UPDATE ... RETURNING` 是 PostgreSQL 很方便的特性。
- 可以减少“先更新、再额外查一次”的往返。

### `DeleteDocument(ctx, input)`

作用：软删除文档。

关键代码块：

1. `UPDATE documents SET is_deleted = true ...`
2. `RowsAffected()` 判断是否真的更新到了记录。
3. 没更新到就返回 `ErrNotFound`。

### `RestoreDocument(ctx, documentID, actorID)`

作用：取消软删除标记。

说明：

- 写法与 `DeleteDocument` 基本对称。
- 这类“成对操作”很适合对比学习。

## 23. `internal/repository/postgres/version_workflow.go`

这是整个项目里最值得学习“事务”概念的文件之一。

### `type VersionWorkflow` 与 `NewVersionWorkflow`

作用：保存数据库连接，并提供构造函数。

### `CreateUploadedVersion(ctx, input)`

作用：在一个事务里完成“新版本入库 + 更新文档当前版本 + 写审计事件”。

关键代码块：

1. `tx, err := w.db.BeginTx(ctx, nil)`
   - 开启事务。

2. `defer tx.Rollback()`
   - 这是 Go 里非常经典的事务保护写法。
   - 即使后面成功 `Commit`，多余的 `Rollback` 也会安全失败，不影响结果。

3. `versionRepo := NewVersionRepository(tx)`
   - 重点：把 `tx` 传给 repository，而不是直接传 `db`。
   - 这样后续 SQL 才能全部在同一个事务里执行。

4. `versionRepo.CreateVersion(...)`
   - 先写版本记录。

5. `UPDATE documents ...`
   - 同步更新文档当前版本和状态。

6. 构造 `extraData`
   - 把文件名、对象 key、存储提供者打包成 JSON，供审计表使用。

7. `shared.RequestIDFromContext(ctx)`
   - 从上下文里拿 request ID，写入审计事件。
   - 这说明前面 middleware 写入的 request ID 已经一路传到了数据层。

8. `INSERT INTO audit_events ...`
   - 记录“替换版本”动作。

9. `tx.Commit()`
   - 只有全部成功才提交事务。

学习点：

- 事务不是数据库层专属概念，它通常服务于业务动作的一致性。
- 这个文件很好地展示了“一次业务动作需要改多张表”的场景。

## 24. `internal/storage/contracts.go`

这个文件定义了文件存储抽象。

### `PutObjectInput`

作用：描述上传文件时需要的输入。

重点字段：

- `ObjectKey`：对象路径。
- `Reader`：文件内容流。
- `Overwrite`：是否允许覆盖。
- `CreatePaths`：是否自动创建父目录。

### `PutObjectResult`

作用：描述上传完成后的结果。

重点字段：

- `ObjectKey`
- `Provider`

### `GetObjectOutput`

作用：描述下载文件得到的结果。

重点字段：

- `Reader`
- `ContentType`
- `Size`

### `FileInfo` 与 `ShareLinkResult`

作用：分别描述文件元信息和共享链接信息。

### `type Provider interface`

这是整个文件最重要的部分。

它规定存储后端必须提供：

- `PutObject`
- `GetObject`
- `DeleteObject`
- `Stat`
- `ListDir`
- `CreateFolder`
- `CreateShareLink`

学习点：

- 业务层不应该直接拼群晖 API，也不应该直接依赖某种文件系统实现。
- 先定义 `Provider` 接口，再由 memory / synology 去实现，是这个项目存储层最重要的设计点。

## 25. 读这套 Go 代码时要重点观察什么

读任意一个函数时，都建议问自己 6 个问题：

1. 输入是什么。
2. 输出是什么。
3. 它依赖谁。
4. 它做的是协议适配、业务编排还是数据执行。
5. 出错时在哪一层被转换语义。
6. 它有没有修改数据库、文件存储、任务状态或审计记录。

如果这 6 个问题都能回答出来，这个函数你基本就读明白了。

## 26. 给 Go 初学者的重点关注项

建议你重点观察这些语言点在本项目里的落点：

1. `if err != nil`
   - 几乎每层都有，是 Go 最基本的错误处理习惯。

2. `context.Context`
   - 从 HTTP 请求一路传到 repository 和事务工作流。

3. interface
   - 在 `repository/contracts.go` 和 `storage/contracts.go` 里最明显。

4. 指针和值接收者
   - 当前很多 service/handler 用值接收者；这说明它们更多是在组合依赖，不太依赖对象内部可变状态。

5. `defer`
   - 在文件关闭、`rows.Close()`、事务 `Rollback()` 中都能看到。

6. 标准库优先
   - 这个项目大量依赖 `net/http`、`database/sql`、`encoding/json`，非常适合初学者建立扎实基础。

## 27. 最后给你的实操建议

如果你想真正把这套代码读懂，不建议一上来全看完。更有效的做法是按链路读：

1. 登录链路
   - `router -> AuthHandler.Login -> AuthService.Login -> UserAuthRepository -> TokenService`

2. 创建文档链路
   - `router -> DocumentHandler.Create -> DocumentService.CreateWithFirstVersion -> DocumentRepository.CreateDocument -> storage.PutObject -> VersionWorkflow.CreateUploadedVersion`

3. 上传版本链路
   - `router -> VersionHandler.Upload -> VersionService.UploadAndCreateVersion -> storage.PutObject -> VersionWorkflow.CreateUploadedVersion`

4. AI 问答链路
   - `router -> AssistantHandler -> AssistantService.Ask -> QueueTask -> AssistantRepository/CreateAssistantRequest -> publisher.Publish`

你可以一次只挑一条链路，从入口一路点进去。这样比按目录机械扫文件更容易建立整体感。

---

如果后续你愿意继续深入，下一步最值得补的是：

1. 对 `handlers/assistant.go` 做同样粒度的函数解释。
2. 对 `repository/postgres/version_repository.go` 做版本 SQL 细读。
3. 结合 `go test ./...`，边看测试边回读 service，会更容易理解输入输出契约。