# Sub2API 二开版：自行构建 Docker 镜像与升级

本文适用于当前二开项目和 `43.128.135.174` 服务器。所有备份均使用 `blaire` 命名，以下命令不清理备份、镜像或数据目录。

## 1. 当前项目和部署信息

| 项目 | 当前值 |
| --- | --- |
| 代码仓库 | https://github.com/18519338432/sub2api |
| 二开分支 | `blaire/subscription-daily-reset` |
| 首次功能提交 | `d98d26190` |
| 服务器登录 | `ssh -i "C:\wyj.pem" ubuntu@43.128.135.174` |
| 服务器架构 | Linux amd64 |
| 部署目录 | `/home/ubuntu/sub2api` |
| Compose 文件 | `/home/ubuntu/sub2api/docker-compose.yml` |
| 应用服务名、容器名 | `sub2api` |
| 数据库容器名 | `sub2api-postgres` |
| 应用数据挂载 | `/home/ubuntu/sub2api/data` → `/app/data` |
| 应用健康接口 | `http://127.0.0.1:18080/health` |
| 已部署的二开镜像 | `blaire/sub2api:daily-reset-d98d26190` |

构建过程为：前端 Vue 构建 → 前端资源嵌入 Go 程序 → 编译 Linux 程序 → 生成运行镜像。

根目录的 `Dockerfile` 已包含以上步骤。使用它构建时，不需要在宿主机单独安装 Go、Node.js 或 pnpm；需要 Docker、Buildx 和 Git。

**修改源码后，需要重新构建镜像，再更新应用容器。仅重启容器不会应用源码改动。**

当前服务器约 3.4 GB 内存。完整源码构建会占用较多内存，优先使用下面的 Windows 本机构建流程；服务器构建作为另一种选择。

## 2. 推荐：在 Windows 本机构建

在 Docker Desktop 中启动 Linux 容器引擎，然后打开 PowerShell。

先把需要发布的源码修改提交到 Git。下面通过 `git archive HEAD` 创建独立构建目录，**只构建当前 HEAD 中已提交的文件**；工作区中未提交的修改不会包含进去。这样也不会把本机 `.cache`、工具链或私钥带入构建目录。

在项目目录执行：

```powershell
Set-Location 'C:\IdeaWorkSpace\中转站维修'

docker version
docker buildx version
docker info --format '{{.OSType}}'
git status --short
git log -1 --oneline
```

`docker info` 的输出应为 `linux`。确认 HEAD 就是要发布的代码后，执行构建：

```powershell
$releaseStamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$releaseCommit = (git rev-parse HEAD).Trim()
$releaseShort = (git rev-parse --short=9 HEAD).Trim()
$releaseDate = [DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ')

# 每次发布自行调整版本号；镜像标签使用时间和提交号，方便定位和回退。
$releaseVersion = '0.2.18-blaire.2'
$releaseImage = "blaire/sub2api:${releaseStamp}-${releaseShort}"

$buildDirectory = Join-Path $env:TEMP "blaire-sub2api-build-$releaseStamp"
$sourceArchive = Join-Path $env:TEMP "blaire-sub2api-source-$releaseStamp.tar"
New-Item -ItemType Directory -Path $buildDirectory | Out-Null

git archive --format=tar --output="$sourceArchive" HEAD
if ($LASTEXITCODE -ne 0) { throw '导出源码失败' }
tar -xf "$sourceArchive" -C "$buildDirectory"
if ($LASTEXITCODE -ne 0) { throw '解压源码失败' }

docker buildx build `
  --platform linux/amd64 `
  --load `
  --build-arg "VERSION=$releaseVersion" `
  --build-arg "COMMIT=$releaseCommit" `
  --build-arg "DATE=$releaseDate" `
  --tag "$releaseImage" `
  --file "$buildDirectory/Dockerfile" `
  "$buildDirectory"

if ($LASTEXITCODE -ne 0) { throw 'Docker 镜像构建失败，请勿继续发布' }
docker image inspect "$releaseImage" --format '{{.Id}}'
Write-Host "本次镜像：$releaseImage"
```

参数含义：

- `--platform linux/amd64`：生成这台 Ubuntu 服务器可用的镜像。
- `--load`：把构建结果加载到本机 Docker；不会自动上传到镜像仓库。
- `VERSION`：程序显示的版本号。
- `COMMIT`：程序显示的源码提交号。
- 最后一个目录：Docker 的构建上下文，必须包含根目录 Dockerfile、前后端源码和 `docs/legal`。

首次构建需要下载基础镜像和依赖，后续构建会复用缓存。

### 导出镜像并上传服务器

继续在同一个 PowerShell 窗口执行，以保留上面的变量：

```powershell
$imageArchive = Join-Path $env:TEMP "blaire-sub2api-image-$releaseStamp.tar"
docker save --output "$imageArchive" "$releaseImage"
if ($LASTEXITCODE -ne 0) { throw '导出镜像失败' }

scp -i 'C:\wyj.pem' "$imageArchive" ubuntu@43.128.135.174:/home/ubuntu/
if ($LASTEXITCODE -ne 0) { throw '上传镜像失败' }

Write-Host "服务器镜像文件：/home/ubuntu/$(Split-Path $imageArchive -Leaf)"
Write-Host "服务器应使用的镜像名：$releaseImage"
```

记录输出的镜像文件路径和镜像名。随后登录服务器：

```powershell
ssh -i 'C:\wyj.pem' ubuntu@43.128.135.174
```

在服务器执行以下命令，把示例文件名和镜像名替换为刚才的实际输出：

```bash
sudo docker load --input /home/ubuntu/blaire-sub2api-image-20261027-103000.tar

IMAGE='blaire/sub2api:20261027-103000-d98d26190'
sudo docker image inspect "$IMAGE" --format '{{.Id}}'
```

镜像加载完成后，从第 4 节开始备份、升级。

## 3. 另一种方式：直接在 Ubuntu 服务器构建

如果已完成第 2 节，不需要重复执行本节。

登录服务器后，在 Bash 中执行。每次创建新的源码目录，保留旧目录：

```bash
set -euo pipefail

RELEASE_STAMP=$(TZ=Asia/Shanghai date +%Y%m%d-%H%M%S)
SOURCE_DIR="/home/ubuntu/blaire-builds/sub2api-$RELEASE_STAMP"
mkdir -p /home/ubuntu/blaire-builds

git clone --depth 1 --branch blaire/subscription-daily-reset \
  https://github.com/18519338432/sub2api.git "$SOURCE_DIR"

cd "$SOURCE_DIR"
git log -1 --oneline

RELEASE_COMMIT=$(git rev-parse HEAD)
RELEASE_SHORT=$(git rev-parse --short=9 HEAD)
RELEASE_VERSION='0.2.18-blaire.2'
IMAGE="blaire/sub2api:$RELEASE_STAMP-$RELEASE_SHORT"

sudo docker buildx build \
  --platform linux/amd64 \
  --load \
  --build-arg "VERSION=$RELEASE_VERSION" \
  --build-arg "COMMIT=$RELEASE_COMMIT" \
  --build-arg "DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --tag "$IMAGE" \
  --file Dockerfile \
  .

sudo docker image inspect "$IMAGE" --format '{{.Id}}'
printf '本次镜像：%s\n' "$IMAGE"
```

本节从 GitHub 指定分支下载代码，然后在服务器构建；构建工作并未发生在 GitHub Actions 中。

## 4. 上线前备份

以下命令在服务器 Bash 中执行。确认变量 `IMAGE` 是本次新镜像名；若重新登录了 SSH，需要重新设置该变量。

```bash
set -euo pipefail
umask 077
cd /home/ubuntu/sub2api

sudo docker image inspect "$IMAGE" --format '{{.Id}}'
df -h /home/ubuntu

BACKUP_STAMP=$(TZ=Asia/Shanghai date +%Y%m%d-%H%M%S)
BACKUP_DIR="/home/ubuntu/blaire-backups/sub2api-$BACKUP_STAMP"
ROLLBACK_IMAGE="blaire/sub2api:rollback-$BACKUP_STAMP"
mkdir -p "$BACKUP_DIR"

# 备份完整数据库，并检查备份能否读取。
sudo docker exec sub2api-postgres sh -c \
  'exec pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Fc' \
  > "$BACKUP_DIR/database.dump"
test -s "$BACKUP_DIR/database.dump"
sudo docker exec -i sub2api-postgres pg_restore --list \
  < "$BACKUP_DIR/database.dump" > "$BACKUP_DIR/database-manifest.txt"

# 备份部署文件、配置和应用数据。
cp -p docker-compose.yml "$BACKUP_DIR/docker-compose.yml"
tar -czf "$BACKUP_DIR/config-and-data.tar.gz" docker-compose.yml .env data
sudo docker inspect sub2api > "$BACKUP_DIR/container-inspect.json"

# 保存当前容器的实际程序，兼容曾在容器内替换过程序的情况。
# 挂载目录中的数据库和应用数据由上面的独立备份保存。
sudo docker commit --pause=false sub2api "$ROLLBACK_IMAGE"

printf 'ROLLBACK_IMAGE=%s\n' "$ROLLBACK_IMAGE" > "$BACKUP_DIR/release.env"
printf '%s\n' "$BACKUP_DIR" > /home/ubuntu/sub2api/blaire-last-backup.txt
printf '备份目录：%s\n回退镜像：%s\n' "$BACKUP_DIR" "$ROLLBACK_IMAGE"
ls -lh "$BACKUP_DIR/database.dump" "$BACKUP_DIR/config-and-data.tar.gz"
```

确认以上步骤全部成功后再升级。备份包含敏感配置，保留目录的访问限制，不要提交到 GitHub。

## 5. 使用新镜像升级应用

仍在服务器 `/home/ubuntu/sub2api` 目录中执行：

```bash
printf '即将使用的新镜像：%s\n' "$IMAGE"
nano docker-compose.yml
```

找到 `services` 下的 `sub2api`，把它的 `image` 改成上面输出的完整镜像名。例如：

```yaml
services:
  sub2api:
    image: blaire/sub2api:20261027-103000-d98d26190
    container_name: sub2api
```

保存后执行：

```bash
# 确认输出中应用镜像是本次 IMAGE 对应的值。
sudo docker compose config --images

# 重建应用容器，继续使用原有 PostgreSQL、Redis 和挂载目录。
sudo docker compose up -d --no-deps sub2api

# 等待启动完成，最多尝试约一分钟。
READY=0
for attempt in $(seq 1 30); do
  if curl --fail --silent --max-time 3 http://127.0.0.1:18080/health; then
    READY=1
    break
  fi
  sleep 2
done
test "$READY" -eq 1

sudo docker compose ps sub2api
sudo docker exec sub2api /app/sub2api --version
sudo docker inspect sub2api \
  --format 'image={{.Config.Image}} status={{.State.Status}} health={{.State.Health.Status}}'
curl --fail --silent --insecure https://43.128.135.174/health
```

确认程序版本、提交号与本次构建相符，健康接口返回 `{"status":"ok"}`。容器健康检查可能稍晚于应用启动，稍候再次检查应显示 `healthy`。随后刷新网页，确认新增功能入口正常。

如果服务启动失败，查看近期日志：

```bash
sudo docker compose logs --tail 100 sub2api
```

新程序启动时会自动执行尚未应用的数据库迁移。本次订阅重置功能对应 `249_subscription_daily_quota_resets.sql`，只新增重置流水表。

## 6. 应用版本回退

发生程序故障时，使用上线前保存的容器镜像恢复程序。先从记录中找出备份目录：

```bash
set -euo pipefail
cd /home/ubuntu/sub2api

BACKUP_DIR=$(cat blaire-last-backup.txt)
source "$BACKUP_DIR/release.env"
sudo docker image inspect "$ROLLBACK_IMAGE" --format '{{.Id}}'

# 保存失败版本的部署配置，再恢复升级前的配置。
FAILED_STAMP=$(TZ=Asia/Shanghai date +%Y%m%d-%H%M%S)
cp -p docker-compose.yml "$BACKUP_DIR/docker-compose.failed-$FAILED_STAMP.yml"
cp -p "$BACKUP_DIR/docker-compose.yml" docker-compose.yml

printf '请将 sub2api 的 image 改为：%s\n' "$ROLLBACK_IMAGE"
nano docker-compose.yml
```

把 `sub2api.image` 改成输出的 `ROLLBACK_IMAGE`，保存后执行：

```bash
sudo docker compose config --images
sudo docker compose up -d --no-deps sub2api
sudo docker exec sub2api /app/sub2api --version
curl --fail --silent http://127.0.0.1:18080/health
```

本次新增流水表与旧程序兼容，回退程序时可以保留该表。若未来版本包含不兼容的数据库变更，应单独制定数据库恢复步骤；直接恢复旧数据库备份会覆盖备份之后产生的业务数据。

## 7. 后续二开和官方升级

- 每次发布使用新的镜像标签，记录版本号、Git 提交号和备份目录。
- 官方发布新版本时，先把官方源码合并到二开分支、处理冲突，再按本文重新构建。
- 换成官方镜像，或使用会覆盖程序的官方在线更新功能，可能覆盖二开功能。
- 目前没有配置 GitHub 自动构建或镜像自动发布。推送源码到 GitHub，不会自动更新服务器镜像。
- 如果以后需要 GitHub 自动构建并发布到 GHCR，需要另外配置 GitHub Actions；本文命令完成的是手动构建和发布。
