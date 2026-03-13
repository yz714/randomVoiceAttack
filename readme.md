# Random Voice Attack

一个低频声音触发的音频播放器，通过麦克风检测低频噪声并自动播放音频文件。

## 功能特性

- 🎤 实时音频检测 - 通过麦克风监听环境声音
- 🎵 智能触发 - 检测到低频噪声时自动播放音频
- 📊 Web界面 - 提供实时监控和统计界面
- 📝 日志记录 - 完整的检测和播放日志
- 💾 数据持久化 - 自动保存噪声数据
- 🛡️ 稳定可靠 - 支持长时间7×24小时运行

## 系统要求

- Windows 10/11
- Go 1.20+ (仅编译时需要)
- 麦克风设备
- 音频文件（MP3, WAV等格式）

## 安装

### 从源码编译

```bash
git clone <repository-url>
cd randomVoiceAttack
go build -o randomVoiceAttack.exe .
```

### 直接使用

下载预编译的 `randomVoiceAttack.exe` 可执行文件。

## 配置

首次运行前，创建 `config.json` 配置文件：

```json
{
  "audio_directory": "res",
  "http_port": 9090,
  "play_count": 3,
  "debug": false,
  "volume_threshold": 0.005,
  "low_freq_ratio_threshold": 0.015,
  "total_energy_threshold": 0.01,
  "anti_lock_interval": 30,
  "bluetooth_heartbeat_interval": 30,
  "play_interval": 60
}
```

### 配置说明

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `audio_directory` | 音频文件目录 | `res` |
| `http_port` | Web服务端口 | `9090` |
| `play_count` | 每次触发播放的音频数量 | `3` |
| `debug` | 调试模式 | `false` |
| `volume_threshold` | 音量阈值 | `0.005` |
| `low_freq_ratio_threshold` | 低频比例阈值 | `0.015` |
| `total_energy_threshold` | 总能量阈值 | `0.01` |
| `anti_lock_interval` | 防止锁屏间隔（秒） | `30` |
| `bluetooth_heartbeat_interval` | 蓝牙心跳间隔（秒） | `30` |
| `play_interval` | 播放间隔（秒） | `60` |

## 使用方法

### 1. 准备音频文件

将你的音频文件放入 `audio_directory` 配置的目录中（默认为 `res`）。

### 2. 启动程序

```bash
# 使用默认配置
randomVoiceAttack.exe

# 指定配置文件
randomVoiceAttack.exe -config myconfig.json

# 启用调试模式
randomVoiceAttack.exe -debug
```

### 3. 访问Web界面

打开浏览器访问：`http://localhost:9090`

Web界面提供：
- 实时噪声数据图表
- 检测日志
- 播放控制按钮（随机播放、连续播放、停止）

### 4. 停止程序

按 `Ctrl+C` 停止程序（按两次强制退出）。

## 目录结构

```
randomVoiceAttack/
├── res/                # 音频文件目录
├── records/            # 噪声录音文件
├── logs/               # 日志文件
├── frontend/           # Web前端文件
├── config.json         # 配置文件
├── noise_data.json     # 噪声数据文件
└── randomVoiceAttack.exe
```

## 技术栈

- **后端**: Go 1.20+
- **音频处理**: go-beep, gonum
- **Web框架**: 标准库 net/http
- **日志**: Zap + Lumberjack
- **前端**: 原生 HTML/CSS/JavaScript + Chart.js

## 常见问题

### Q: 检测太灵敏/不灵敏怎么办？

A: 调整 `config.json` 中的以下参数：
- 降低 `VolumeThreshold` 使检测更灵敏
- 提高 `VolumeThreshold` 使检测更迟钝
- 调整 `LowFreqRatioThreshold` 改变低频检测敏感度

### Q: 音频文件没有被检测到？

A: 确保：
- 音频文件放在正确的目录
- 音频格式支持（MP3, WAV等）
- 检查日志看是否有错误信息

### Q: 如何重置所有数据？

A: 删除以下文件：
- `noise_data.json` - 噪声数据
- `logs/` 目录 - 日志文件
- `records/` 目录 - 录音文件（可选）

## 开发

### 运行测试

```bash
# 运行所有测试
go test -v ./...

# 运行特定包测试
go test -v ./config
go test -v ./detector
```

### 代码结构

```
randomVoiceAttack/
├── config/         # 配置管理
├── controller/     # 播放控制
├── detector/       # 音频检测
├── logger/         # 日志系统
├── player/         # 音频播放
├── services/       # HTTP服务和心跳
└── utils/          # 工具函数
```

## 版本历史

### v0.8.0
- 完善用户文档（README.md）
- 修正配置文件示例和说明
- 更新默认配置值（res目录、9090端口）
- 准备v1.0.0发布

### v0.7.3
- 修复全局变量并发安全
- 优化魔法数字使用
- 改进命名规范

### v0.7.2
- 改进错误处理
- 添加类型断言验证

### v0.7.1
- 修复切片内存泄漏
- 实现records目录文件管理
- 优化通道缓冲区
- 修复goroutine泄漏

### v0.7.0
- 增加单元测试覆盖
- 配置模块完整测试

### v0.6.1
- 完善代码注释
- 添加Godoc风格文档

### v0.5.0
- 添加接口定义
- 降低模块耦合度

### v0.4.0
- 性能优化（减少磁盘I/O）
- 周期性保存数据

### v0.3.0
- 前端布局优化
- 实现停止播放功能

### v0.2.1
- Windows兼容性修复
- 原子文件写入

### v0.2.0
- 代码质量大幅提升
- 多项bug修复

### v0.1.0
- 初始版本
- 基础功能实现

## 许可证

本项目仅供学习和个人使用。

## 贡献

欢迎提交Issue和Pull Request！

## 致谢

感谢所有为本项目做出贡献的开发者。
