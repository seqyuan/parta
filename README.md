# parta

parallel task


## Update
- 2025-05-05 项目初始版本发布  
- 2025-05-08 修复import语句位置导致的编译错误，优化项目结构

## 使用说明
### 编译
```bash
go build -o parta ./cmd/app/main.go
```

### 运行
```bash
./parta [参数]
```

### 参数说明
#### 基本参数
- `--config string`  
  指定配置文件路径 (默认: "./config.yaml")
- `--verbose`  
  启用详细日志模式 (默认: false)

#### 任务控制参数

#### 输出控制参数


## 版本管理
`git add -A && git commit -m "v1.0.15" && git push && git tag v1.0.15 && git push origin v1.0.15`
