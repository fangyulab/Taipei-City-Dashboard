# 資料庫同步指南 Database Sync Guide

## 匯出（更新到 GitHub）

當你修改了資料庫（新增組件、改配置等），需要匯出並推到 GitHub 讓其他人同步。

### Windows（PowerShell）

```powershell
cd C:\Users\你的帳號\Documents\GitHub\Taipei-City-Dashboard\docker

# 匯出兩個資料庫（加 --clean 自動產生 DROP 語句）
docker exec postgres-manager pg_dump -U postgres --clean --no-owner dashboardmanager > sql/dashboardmanager_backup.sql
docker exec postgres-data pg_dump -U postgres --clean --no-owner dashboard > sql/dashboard_backup.sql

# 推到 GitHub
git add sql/
git commit -m "sync db"
git push
```

### macOS / Linux（Terminal）

```bash
cd Taipei-City-Dashboard/docker

# 匯出兩個資料庫（加 --clean 自動產生 DROP 語句）
docker exec postgres-manager pg_dump -U postgres --clean --no-owner dashboardmanager > sql/dashboardmanager_backup.sql
docker exec postgres-data pg_dump -U postgres --clean --no-owner dashboard > sql/dashboard_backup.sql

# 推到 GitHub
git add sql/
git commit -m "sync db"
git push
```

---

## 匯入（從 GitHub 同步到本地）

拉取最新的 DB dump 並匯入到本地資料庫。**匯入會覆蓋現有資料。**

### Windows（用 cmd，不要用 PowerShell）

> ⚠️ 必須用 **cmd**（按 Win+R 輸入 cmd），PowerShell 有編碼問題會導致中文變亂碼。

**方法一（推薦）：複製進容器執行，完全避免編碼問題**

```cmd
cd C:\Users\你的帳號\Documents\GitHub\Taipei-City-Dashboard\docker

git pull

:: 先清空再匯入 dashboardmanager
docker exec postgres-manager psql -U postgres -d dashboardmanager -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
docker exec postgres-manager psql -U postgres -d dashboardmanager -c "CREATE EXTENSION IF NOT EXISTS postgis;"
docker cp sql\dashboardmanager_backup.sql postgres-manager:/tmp/
docker exec postgres-manager psql -U postgres -d dashboardmanager -f /tmp/dashboardmanager_backup.sql

:: 先清空再匯入 dashboard
docker exec postgres-data psql -U postgres -d dashboard -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
docker exec postgres-data psql -U postgres -d dashboard -c "CREATE EXTENSION IF NOT EXISTS postgis;"
docker cp sql\dashboard_backup.sql postgres-data:/tmp/
docker exec postgres-data psql -U postgres -d dashboard -f /tmp/dashboard_backup.sql

:: 重啟後端和前端
docker-compose restart dashboard-be
docker-compose restart dashboard-fe
```

**方法二：用 cmd 的 type 指令**

```cmd
cd C:\Users\你的帳號\Documents\GitHub\Taipei-City-Dashboard\docker

git pull

:: 設定 UTF-8 編碼
chcp 65001

:: 先清空再匯入 dashboardmanager
docker exec postgres-manager psql -U postgres -d dashboardmanager -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
docker exec postgres-manager psql -U postgres -d dashboardmanager -c "CREATE EXTENSION IF NOT EXISTS postgis;"
type sql\dashboardmanager_backup.sql | docker exec -i postgres-manager psql -U postgres -d dashboardmanager

:: 先清空再匯入 dashboard
docker exec postgres-data psql -U postgres -d dashboard -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
docker exec postgres-data psql -U postgres -d dashboard -c "CREATE EXTENSION IF NOT EXISTS postgis;"
type sql\dashboard_backup.sql | docker exec -i postgres-data psql -U postgres -d dashboard

:: 重啟後端和前端
docker-compose restart dashboard-be
docker-compose restart dashboard-fe
```

### macOS / Linux（Terminal）

```bash
cd Taipei-City-Dashboard/docker

git pull

# 先清空再匯入 dashboardmanager
docker exec postgres-manager psql -U postgres -d dashboardmanager -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
docker exec postgres-manager psql -U postgres -d dashboardmanager -c "CREATE EXTENSION IF NOT EXISTS postgis;"
docker exec -i postgres-manager psql -U postgres -d dashboardmanager < sql/dashboardmanager_backup.sql

# 先清空再匯入 dashboard
docker exec postgres-data psql -U postgres -d dashboard -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
docker exec postgres-data psql -U postgres -d dashboard -c "CREATE EXTENSION IF NOT EXISTS postgis;"
docker exec -i postgres-data psql -U postgres -d dashboard < sql/dashboard_backup.sql

# 重啟後端和前端
docker-compose restart dashboard-be
docker-compose restart dashboard-fe
```

---

## 如果新增了 npm 套件

有人新增了前端套件（例如 mapbox-gl-draw），同步後還需要：

### Windows / macOS

```bash
docker exec dashboard-fe npm install
docker-compose restart dashboard-fe
```

---

## 如果新增了 geojson 檔案

geojson 檔案跟著前端程式碼放在 GitHub，`git pull` 後需要重啟前端：

```bash
docker-compose restart dashboard-fe
```

---

## 一鍵同步腳本

### Windows — `sync.bat`（放在 docker 資料夾，用 cmd 執行）

```bat
@echo off
chcp 65001 >nul
echo [1/4] Pulling latest code...
git pull
echo [2/4] Syncing dashboardmanager DB...
docker exec postgres-manager psql -U postgres -d dashboardmanager -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
docker exec postgres-manager psql -U postgres -d dashboardmanager -c "CREATE EXTENSION IF NOT EXISTS postgis;"
docker cp sql\dashboardmanager_backup.sql postgres-manager:/tmp/
docker exec postgres-manager psql -U postgres -d dashboardmanager -f /tmp/dashboardmanager_backup.sql
echo [3/4] Syncing dashboard DB...
docker exec postgres-data psql -U postgres -d dashboard -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
docker exec postgres-data psql -U postgres -d dashboard -c "CREATE EXTENSION IF NOT EXISTS postgis;"
docker cp sql\dashboard_backup.sql postgres-data:/tmp/
docker exec postgres-data psql -U postgres -d dashboard -f /tmp/dashboard_backup.sql
echo [4/4] Restarting services...
docker-compose restart dashboard-be
docker-compose restart dashboard-fe
echo Done!
```

### macOS — `sync.sh`（放在 docker 資料夾）

```bash
#!/bin/bash
set -e
echo "[1/4] Pulling latest code..."
git pull
echo "[2/4] Syncing dashboardmanager DB..."
docker exec postgres-manager psql -U postgres -d dashboardmanager -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
docker exec postgres-manager psql -U postgres -d dashboardmanager -c "CREATE EXTENSION IF NOT EXISTS postgis;"
docker exec -i postgres-manager psql -U postgres -d dashboardmanager < sql/dashboardmanager_backup.sql
echo "[3/4] Syncing dashboard DB..."
docker exec postgres-data psql -U postgres -d dashboard -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
docker exec postgres-data psql -U postgres -d dashboard -c "CREATE EXTENSION IF NOT EXISTS postgis;"
docker exec -i postgres-data psql -U postgres -d dashboard < sql/dashboard_backup.sql
echo "[4/4] Restarting services..."
docker-compose restart dashboard-be
docker-compose restart dashboard-fe
echo "Done!"
```

使用前先給權限：`chmod +x sync.sh`

---

## 注意事項

- **匯入會覆蓋所有現有資料**，匯入前確認自己的改動已匯出並推到 GitHub
- **多人同時改 DB 會衝突**，建議改之前先溝通，避免互相覆蓋
- **Windows 請用 cmd 不要用 PowerShell**，PowerShell 的編碼會把中文弄成亂碼
