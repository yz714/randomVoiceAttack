# Random Voice Attack

一个基于低频声音触发的随机音频播放器。

## 功能特性

- **低频声音检测**：通过FFT快速傅里叶变换实时检测环境中的低频噪音
- **随机音频播放**：检测到低频噪音后，随机播放指定目录下的音频文件
- **Web监控界面**：提供实时监控界面，展示音频能量图、低频比率曲线和检测日志
- **优雅退出机制**：支持Ctrl+C优雅退出，再次按Ctrl+C可强制退出
- **蓝牙心跳功能**：定期播放静音音频保持蓝牙连接活跃
- **防锁屏功能**：定期模拟用户活动防止系统锁屏

## 系统要求

- Windows操作系统
- Go 1.16+

## 安装和运行

### 编译

```bash
go build
```

### 运行

```bash
./randomVoiceAttack
```

### 命令行参数

```
  -c, --config string   Path to config file (default "config.json")
  -d, --debug           Enable debug mode (shorthand)
      --debug           Enable debug mode
  -h, --help            Show help message (shorthand)
      --help            Show help message
```

## 配置文件

配置文件 `config.json` 包含以下选项：

```json
{
  "play_count": 3,
  "anti_lock_interval": 30,
  "bluetooth_heartbeat_interval": 30,
  "play_interval": 60,
  "audio_directory": "res",
  "http_port": 9090,
  "debug": false,
  "volume_threshold": 0.005,
  "low_freq_ratio_threshold": 0.015,
  "total_energy_threshold": 0.01
}
```

### 配置说明

- `play_count`: 每次检测到噪音后播放的音频数量
- `anti_lock_interval`: 防锁屏间隔（秒）
- `bluetooth_heartbeat_interval`: 蓝牙心跳间隔（秒）
- `play_interval`: 检测间隔（秒）
- `audio_directory`: 音频文件目录
- `http_port`: Web服务器端口
- `debug`: 调试模式开关
- `volume_threshold`: 音量检测阈值
- `low_freq_ratio_threshold`: 低频比率阈值
- `total_energy_threshold`: 总能量阈值

## Web界面

启动程序后，访问 `http://localhost:9090` 可以查看实时监控界面，包含：

- 音频能量条可视化
- 实时检测日志终端
- 低频比率直方图
- 音量时间序列曲线
- 播放状态统计

## 项目结构

```
randomVoiceAttack/
├── config/           # 配置模块
├── controller/       # 音频控制模块
├── detector/         # 低频检测模块
├── frontend/         # 前端界面
├── logger/           # 日志模块
├── logs/             # 日志文件目录
├── player/           # 音频播放模块
├── records/          # 录音文件目录
├── res/              # 音频资源目录
├── services/         # HTTP和心跳服务
├── utils/            # 工具函数
├── config.json       # 配置文件
├── main.go           # 主程序入口
└── README.md         # 项目文档
```

## 技术栈

- **后端**: Go语言
- **音频处理**: beep库 + FFT
- **前端**: HTML5 + Chart.js
- **日志**: Zap + Lumberjack
- **Windows音频**: winmm.dll

## 版本历史

### v0.1.0 (2026-03-13)

- 初始版本发布
- 低频声音检测功能
- 随机音频播放功能
- Web监控界面
- 优雅退出和强制退出机制
- 蓝牙心跳和防锁屏功能
