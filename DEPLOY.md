# PaperMind 部署文档

## 项目架构

| 组件 | 说明 |
|------|------|
| 后端 | Go + Gin，依赖 PostgreSQL(pgvector) + Redis + 阿里云API |
| 前端 | React + Vite，构建为静态文件 |
| 服务器A | 前端静态文件 + Nginx反向代理 (已有SSL/域名) |
| 服务器B | Go后端 + PostgreSQL + Redis (Docker部署) |

---

## 一、服务器B环境准备

### 安装 Docker
```bash
sudo apt update && sudo apt install -y ca-certificates curl gnupg
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt update && sudo apt install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
sudo systemctl start docker && sudo systemctl enable docker
sudo usermod -aG docker <用户名>  # 重新登录生效
```

### 配置镜像加速
```bash
sudo tee /etc/docker/daemon.json <<-'EOF'
{"registry-mirrors": ["https://registry.cn-hangzhou.aliyuncs.com"]}
EOF
sudo systemctl daemon-reload && sudo systemctl restart docker
```

---

## 二、本地准备文件

### 1. 交叉编译后端 (Windows PowerShell)
```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"
go build -o papermind ./cmd/server
```

### 2. 导出 Docker 镜像 (WSL)
```bash
docker pull pgvector/pgvector:pg16
docker pull redis:7-alpine
docker save pgvector/pgvector:pg16 -o pgvector.tar
docker save redis:7-alpine -o redis.tar
```

### 3. 构建前端
```bash
cd web && npm install && npm run build
```

---

## 三、上传到服务器B

```bash
scp papermind pgvector.tar redis.tar docker-compose.yml sql/InitDatabase.sql .env start.sh stop.sh <用户名>@<服务器B_IP>:/home/<用户名>/
```

---

## 四、服务器B部署

### 1. 导入镜像并启动容器
```bash
docker load -i pgvector.tar && docker load -i redis.tar && rm pgvector.tar redis.tar
docker compose up -d
```

### 2. 初始化数据库
```bash
docker exec -i postgres psql -U <数据库用户> -d paper_mind < InitDatabase.sql
```

### 3. 启动后端
```bash
chmod +x papermind start.sh stop.sh
./start.sh
```

---

## 五、服务器A部署前端

### 1. 上传前端文件
```bash
scp -r web/dist/* <用户名>@<服务器A_IP>:/home/<用户名>/PaperMindFrontend/
```

### 2. Nginx配置 (在server块内添加)
```nginx
location /paper_mind/ {
    alias /home/<用户名>/PaperMindFrontend/;
    try_files $uri $uri/ /paper_mind/index.html;
}

location /paper_mind_api/ {
    rewrite ^/paper_mind_api/(.*)$ /api/$1 break;
    proxy_pass http://<服务器B_IP>:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}
```

### 3. 重载Nginx
```bash
sudo nginx -t && sudo nginx -s reload
```

---

## 六、访问验证

访问 `https://<域名>/paper_mind/` 测试注册、登录、上传论文、问答功能。

---

## 运维命令

```bash
# 后端
./start.sh          # 启动
./stop.sh           # 停止
tail -f papermind.log  # 查看日志

# Docker
docker compose ps    # 查看容器状态
docker compose down  # 停止容器
```