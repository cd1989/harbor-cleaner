# Harbor Cleaner

Harbor Cleaner 是一个用于清理 Harbor 中镜像的工具。

## 功能特性

- **安全删除镜像 tag** 解决了同 repo 下内容相同的其他 tag 被删掉的副作用。
- **灵活选择删除策略** 支持多个镜像清理策略，满足不同的业务需要。
- **DryRun** 在真正执行清理前先 DryRun 运行，检查哪些镜像会被清理。
- **定时执行** 可以通过 CRON 表达式配置定期清理镜像。

## 概念

| Image | Project | Repo | Tag |
|--|--|--|--|
| library/busybox:1.30.0 | library | busybox | 1.30.0 |  
| release/devops/tools:v1.0 | release | devops/tools | v1.0 |

## 清理策略

### 保留最新策略

该策略会保留每个镜像最新的 N 个 tag，而清理掉其余的，该策略依照镜像的创建时间判断镜像的新与旧。

```yaml
numberPolicy:
    number: 5
```

上述配置表示保留每个 repo 下最新的 5 个 tag。

### 正则匹配策略

正则匹配策略通过正则表达式匹配需要清理的镜像，提供 repo 名称、tag 名称的匹配。每种类型的匹配都可以配置多个正则表达式，只要其中的一个表达式匹配上即为匹配。repo 和 tag 同时匹配才会触发清理。

正则表达式的语法请参照 Go 语言的正则表达式语法，例如 `.*` 表示匹配所有。

```yaml
regexPolicy:
    repos: [".*"]
    tags: [".*-alpha.*", "dev"]
```

上述配置会清理掉所有 repo 下 tag 名为 `dev` 或者包含 `alpha` 的镜像，例如：

- dev
- v1.0.0-alpha
- v1.4.0-alpha.2
- 1.0-alpha.5

## 最近未被使用策略

该策略用于清理指定时间内未被使用过的镜像，pull, push, delete 均视为对镜像的使用。策略的实现依赖于 Harbor 的操作日志。

```yaml
notTouchedPolicy:
  time: 604800
```

## 使用方法

### 获取镜像

可以从源代码快速构建一个镜像：

```bash
$ make image VERSION=latest
```

也可以从 DockerHub 上拉取一个发布的镜像版本：

```bash
$ docker pull k8sdevops/harbor-cleaner:v0.4.0
```

### 配置

```yaml
# Harbor 地址
host: https://dev.cargo.io
# Harbor 版本，例如 1.7, 1.4.0
version: 1.7
# 拥有管理员权限的账号
auth:
  user: admin
  password: Pwd123456
# 希望清理的项目列表，如果为空表示清理所有的项目
projects: []
# 清理策略配置
policy:
  # 策略类型，支持的策略包含：
  # - "number": 保留最新策略
  # - "regex": 正则匹配策略
  # - "recentlyNotTouched": 最近未被使用策略
  type: number

  # 配置【保留最新策略】，仅当 policy.type 配置为 "number" 时才需要配置，也只有这时才会生效。
  numberPolicy:
    # 保留最新的多少个 Tag
    number: 5

  # 配置【正则匹配策略】，仅当 policy.type 配置为 "regex" 时才需要配置，也只有这时才会生效。
  regexPolicy:
    # 匹配 repo 的正则表达式列表，匹配列表中的任意一个即视为匹配。
    repos: [".*"]
    # 匹配 tag 的正则表达式列表，匹配列表中的任意一个即视为匹配。
    tags: [".*-alpha.*", "dev"]

  # 配置【最近未被使用策略】，仅当 policy.type 配置为 "recentlyNotTouched" 时才需要配置，也只有这时才会生效。
  notTouchedPolicy:
    # 以秒计的时间，表示多久未被使用，超过该时间未被使用的镜像会被清理掉。
    time: 604800

  # 不允许被清理的 tag，支持 '?', '*' 通配符。该配置可以用于保护那些不希望被清理策略清理掉的镜像。
  retainTags: []
# 镜像清理触发器，目前支持 CRON 表达式进行定时触发
trigger:
  # 定时触发 CRON 表达式，例如 "0 0 * * *"。如果不想定期执行，请保留空值。注：配置的 CRON 表达式需要用双引号引起来。
  # 这里 CRON 的时区由运行 harbor-cleaner 的环境决定，通过容器执行的话（docker run），使用的是 UTC 时间。
  cron:
# 对于 Harbor v1.9 以上版本，需要配置 XSRF，对于其他版本直接忽略这部分配置。
xsrf:
  # 对用 Harbor 配置文件 'common/config/core/app.conf' 中 'EnableXSRF' 的值。
  enabled: true
  # 对应 'common/config/core/app.conf' 中 'XSRFKey' 的值。
  key: T20zVqpLbDDlQGVIiiwDtAAtsm8bSRjHBJSMyejG
```

在清理策略配置部分，只需要根据策略类型配置 `numberPolicy`, `regexPolicy`, `notTouchedPolicy` 其一即可。

### 执行 DryRun

```bash
$ docker run -it --rm \
    -v <your-config-file>:/workspace/config.yaml \
    k8sdevops/harbor-cleaner:latest --dryrun=true
```

需要手动创建配置文件并挂载到容器中。

### 执行清理

```bash
$ docker run -it --rm \
    -v <your-config-file>:/workspace/config.yaml \
    k8sdevops/harbor-cleaner:latest
```

需要手动创建配置文件并挂载到容器中。

### 定时触发


```yaml
# 镜像清理触发器，目前支持 CRON 表达式进行定时触发
trigger:
  # 定时触发 CRON 表达式，例如 "0 0 * * *"。如果不想定期执行，请保留空值。注：配置的 CRON 表达式需要用双引号引起来。
  cron: "0 0 * * *"
```

```bash
$ docker run -d --name=harbor-cleaner --rm \
    -v <your-config-file>:/workspace/config.yaml \
    k8sdevops/harbor-cleaner:latest
```

需要手动创建配置文件并挂载到容器中。

## 支持的 Harbor 版本

- 1.4.x
- 1.5.x
- 1.6.x
- 1.7.x
- 1.8.x
- 1.9.x (harbor-cleaner:v0.4.0+)

其他版本可能也支持，但是没有经过测试。
