# dq-pipeline — Go 数据质量评估与清洗流水线 HTTP 服务

本数据质量评估与清洗流水线 HTTP 服务：多阶段评估与清洗走同一套规则，HTTP 与子命令结果必须一致；未知字段或非法阈值不得当成功。

## 构建 / 运行 / 测试

```text
go build ./...     # 编译
go run .
go test ./...      # 测试
```

## 评测镜像

本目录评测专用文件（勿覆盖项目自带 Dockerfile/README）：

- `benzhi.Dockerfile`
- `build_benzhi_docker.sh`
- `BENZHI_README.md`（本文件）

两种架构都要构建并进容器验证：

```bash
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh <image-name> linux/arm64
./build_benzhi_docker.sh <image-name> linux/amd64
docker run -it <image-name>:latest
```
