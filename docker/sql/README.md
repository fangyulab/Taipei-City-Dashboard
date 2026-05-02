# 每次匯出傳上 github
```sh
docker exec postgres-manager pg_dump -U postgres dashboardmanager > sql/dashboardmanager_backup.sql
docker exec postgres-data pg_dump -U postgres dashboard > sql/dashboard_backup.sql
```

# 每次匯入自己本地

```sh
git pull
docker exec -i postgres-manager psql -U postgres -d dashboardmanager < sql/dashboardmanager_backup.sql
docker exec -i postgres-data psql -U postgres -d dashboard < sql/dashboard_backup.sql
```

