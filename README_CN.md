# img2webp

高性能图片转 WebP 转换工具，自动优化压缩质量。

## 功能特性

- **自动质量选择** — 分析图片内容，自动选择最佳编码参数
- **多候选并行编码** — 并行测试多个质量级别，选择最小且满足质量阈值的方案
- **内容感知处理** — 自动识别照片、图形和透明通道，采用相应编码策略
- **嵌入式二进制** — 无外部依赖，`cwebp` 已内嵌支持主流平台
- **跨平台支持** — macOS (ARM64/x64)、Linux (ARM64/x64)、Windows (x64)

## 安装

### Go 安装

```bash
go install github.com/zoulux/img2webp@latest
```

### 下载二进制

从 [Releases 页面](https://github.com/zoulux/img2webp/releases) 下载对应平台的二进制文件。

下载后需要授权才能执行：

```bash
# macOS / Linux
chmod +x img2webp-*

# 然后运行
./img2webp-darwin-arm64  # 以 macOS Apple Silicon 为例
```

> **macOS 用户注意**：首次运行可能会提示安全警告。前往 `系统设置 > 隐私与安全性` 点击"仍要打开"，或者运行：
> ```bash
> xattr -d com.apple.quarantine img2webp-darwin-arm64
> ```

| 平台 | 架构 | 文件 |
|------|------|------|
| macOS | Apple Silicon (M1/M2/M3) | `img2webp-darwin-arm64` |
| macOS | Intel | `img2webp-darwin-amd64` |
| Linux | ARM64 | `img2webp-linux-arm64` |
| Linux | x86_64 | `img2webp-linux-amd64` |
| Windows | x86_64 | `img2webp-windows-amd64.exe` |

### 源码编译

```bash
git clone https://github.com/zoulux/img2webp.git
cd img2webp
go build -o img2webp .
```

## 使用方法

### 基本用法

```bash
# 转换当前目录下所有图片
img2webp

# 转换指定文件
img2webp photo.jpg

# 转换目录下所有图片
img2webp /path/to/images
```

### 输出目录

```bash
# 指定输出目录（默认为 "output"）
img2webp -o /path/to/output photos/
img2webp --output ./webp-images .
```

### 质量控制

```bash
# 自动质量（默认，推荐）
img2webp -q 0 images/

# 固定质量
img2webp -q 85 photo.png
img2webp --quality 90 *.jpg
```

### 编码模式

```bash
# 自动检测（默认）— 自动判断最佳模式
img2webp --mode auto images/

# 照片模式 — 针对照片优化
img2webp --mode photo photos/

# 图形模式 — 针对截图、UI 元素优化
img2webp --mode graphic screenshots/

# 无损模式 — 精确保留像素
img2webp --mode lossless icons/
```

### 高级选项

```bash
# 覆盖已存在的输出文件
img2webp --overwrite images/

# 并行工作数（默认为 CPU 核心数）
img2webp --workers 8 large-batch/

# 重新编码已有的 WebP 文件
img2webp --reencode-webp images/

# 预览模式 — 不写入文件
img2webp --dry-run images/
```

## 命令参考

```
用法: img2webp [参数] [输入]

参数:
  input                     输入文件或目录（默认 "."）

选项:
  -o, --output string       输出目录（默认 "output"）
  -q, --quality int         质量 1-100，0 表示自动（默认 0）
      --overwrite           覆盖已存在的输出文件
      --workers int         工作线程数，0 表示 CPU 核心数（默认 0）
      --reencode-webp       重新编码输入的 webp 文件
      --mode string         auto|photo|graphic|lossless（默认 "auto"）
      --dry-run             预览模式，不写入文件
  -h, --help                显示帮助
```

## 工作原理

1. **输入收集** — 扫描输入路径，查找支持的图片格式（JPEG、PNG、WebP）

2. **图片分析** — 检测：
   - 图片尺寸和宽高比
   - 透明通道
   - 内容类型（照片 vs 图形）
   - 颜色复杂度

3. **候选生成** — 根据以下条件生成编码候选：
   - 内容分类
   - 质量要求
   - 透明处理需求

4. **并行编码** — 使用工作池并行编码多个质量候选

5. **质量评分** — 使用以下指标评估每个候选：
   - SSIM（结构相似性指数）
   - 边缘保留指标
   - 透明边缘质量（针对透明图片）

6. **选择** — 选择满足质量阈值的最小文件

7. **输出** — 将优化后的 WebP 写入输出目录，保留原始目录结构

## 支持格式

### 输入
- JPEG (.jpg, .jpeg)
- PNG (.png)
- WebP (.webp)

### 输出
- WebP (.webp)

## 性能

工具在多个层级使用并行处理：
- **文件级并行** — 同时处理多个图片
- **候选级并行** — 每张图片并行编码多个质量候选

在典型的现代机器（8+ 核心）上，批量转换可获得显著加速。

## 依赖

- **嵌入式 `cwebp` 二进制** — Google libwebp 编码器已内嵌于所有支持平台，无需外部安装
- **Go 运行时** — 仅在源码编译或使用 `go install` 时需要

## 许可证

MIT License
