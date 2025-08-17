# RealtimeTTS Go

一个高性能的实时文本转语音Go库，采用流式架构设计，支持多种TTS引擎，实现低延迟的语音合成和播放。

## 🚀 主要特性

- **实时性**：基于goroutine的流式处理，边合成边播放
- **多引擎支持**：ESpeak、Azure、OpenAI、Coqui、ElevenLabs等
- **低延迟**：优化的缓冲策略和异步处理
- **高可靠性**：引擎故障自动切换和错误恢复
- **易扩展**：接口化设计，支持新引擎接入
- **Go原生**：充分利用Go语言的并发和内存管理特性

## 📋 系统要求

- Go 1.22 或更高版本
- ESpeak (可选，用于本地TTS)
- 音频设备支持

## 🛠️ 安装

### 1. 作为Go模块依赖

```bash
go get github.com/yourusername/realtimetts
```

### 2. 本地开发

```bash
git clone https://github.com/yourusername/realtimetts.git
cd realtimetts
go mod tidy
```

## 🎯 快速开始

### 基本使用

```go
package main

import (
    "context"
    "time"
    
    "github.com/yourusername/realtimetts"
)

func main() {
    // 创建TTS客户端
    client := realtimetts.NewClient(realtimetts.DefaultConfig())
    
    // 创建上下文
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // 播放文本
    err := client.Speak(ctx, "Hello, this is a test of the RealtimeTTS Go library.")
    if err != nil {
        panic(err)
    }
}
```

### 异步播放

```go
// 异步播放
client.SpeakAsync(ctx, "This is an async playback test.")

// 等待一段时间
time.Sleep(5 * time.Second)

// 停止播放
client.Stop()
```

### 使用自定义回调

```go
callbacks := &realtimetts.Callbacks{
    OnWord: func(word string) {
        fmt.Printf("🎤 %s ", word)
    },
    OnPlaybackStart: func() {
        fmt.Println("▶️ 开始播放")
    },
    OnPlaybackStop: func() {
        fmt.Println("⏹️ 停止播放")
    },
}

err := client.SpeakWithCallbacks(ctx, "Hello world!", callbacks)
```

### 语音代理集成示例

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/yourusername/realtimetts"
)

// VoiceAgent 语音代理
type VoiceAgent struct {
    client *realtimetts.Client
}

// NewVoiceAgent 创建语音代理
func NewVoiceAgent() *VoiceAgent {
    config := realtimetts.LowLatencyConfig()
    client := realtimetts.NewClient(config)
    
    return &VoiceAgent{client: client}
}

// Speak 语音输出
func (va *VoiceAgent) Speak(text string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    callbacks := &realtimetts.Callbacks{
        OnWord: func(word string) {
            fmt.Printf("🎤 %s ", word)
        },
        OnPlaybackStart: func() {
            fmt.Println("\n▶️ 语音代理开始说话")
        },
        OnPlaybackStop: func() {
            fmt.Println("\n⏹️ 语音代理停止说话")
        },
    }
    
    return va.client.SpeakWithCallbacks(ctx, text, callbacks)
}

// ProcessInput 处理用户输入
func (va *VoiceAgent) ProcessInput(input string) {
    fmt.Printf("用户输入: %s\n", input)
    
    // 生成响应
    response := va.generateResponse(input)
    
    // 语音输出
    va.Speak(response)
}

// generateResponse 生成响应
func (va *VoiceAgent) generateResponse(input string) string {
    switch {
    case contains(input, "你好"):
        return "你好！我是语音代理，很高兴为您服务。"
    case contains(input, "天气"):
        return "今天天气不错，适合出门走走。"
    case contains(input, "时间"):
        return fmt.Sprintf("现在时间是 %s", time.Now().Format("15:04:05"))
    case contains(input, "再见"):
        return "再见！祝您有愉快的一天。"
    default:
        return "我理解您的话，请继续。"
    }
}

func contains(s, substr string) bool {
    return len(s) >= len(substr) && (s == substr || 
        (len(s) > len(substr) && (s[:len(substr)] == substr || 
        s[len(s)-len(substr):] == substr || 
        func() bool {
            for i := 1; i <= len(s)-len(substr); i++ {
                if s[i:i+len(substr)] == substr {
                    return true
                }
            }
            return false
        }())))
}

func main() {
    agent := NewVoiceAgent()
    
    inputs := []string{
        "你好",
        "今天天气怎么样？",
        "现在几点了？",
        "再见",
    }
    
    for _, input := range inputs {
        fmt.Printf("\n用户: %s\n", input)
        agent.ProcessInput(input)
        time.Sleep(2 * time.Second)
    }
}
```

## 🔧 配置选项

### 音频配置

```go
// 默认配置
config := realtimetts.DefaultConfig()

// 高质量配置
config := realtimetts.HighQualityConfig()

// 低延迟配置
config := realtimetts.LowLatencyConfig()

// 自定义配置
config := &realtimetts.AudioConfig{
    Format:           realtimetts.FormatPCM,
    Channels:         1,
    SampleRate:       22050,
    Muted:            false,
    FramesPerBuffer:  512,
    PlayoutChunkSize: -1,
}
```

### 回调函数

```go
callbacks := &realtimetts.Callbacks{
    OnCharacter: func(char rune) {
        // 处理每个字符
    },
    OnWord: func(word string) {
        // 处理每个单词
    },
    OnAudioChunk: func(chunk []byte) {
        // 处理音频块
    },
    OnPlaybackStart: func() {
        // 播放开始
    },
    OnPlaybackStop: func() {
        // 播放停止
    },
    OnTextStreamStart: func() {
        // 文本流开始
    },
    OnTextStreamStop: func() {
        // 文本流停止
    },
    OnSentenceSynthesized: func(sentence string) {
        // 句子合成完成
    },
}
```

## 🔌 引擎扩展

### 实现自定义引擎

```go
type CustomEngine struct {
    *realtimetts.BaseEngine
    streamInfo realtimetts.StreamInfo
}

func NewCustomEngine() *CustomEngine {
    config := &realtimetts.AudioConfig{
        Format:      realtimetts.FormatPCM,
        Channels:    1,
        SampleRate:  16000,
        Muted:       false,
    }

    engine := &CustomEngine{
        BaseEngine: realtimetts.NewBaseEngine("CustomEngine", config),
        streamInfo: realtimetts.StreamInfo{
            Format:     realtimetts.FormatPCM,
            Channels:   1,
            SampleRate: 16000,
        },
    }

    return engine
}

func (e *CustomEngine) Synthesize(ctx context.Context, text string) error {
    // 实现自定义合成逻辑
    audioData := e.customSynthesis(text)
    
    select {
    case e.BaseEngine.AudioQueue <- audioData:
    case <-ctx.Done():
        return ctx.Err()
    }
    
    return nil
}

func (e *CustomEngine) GetStreamInfo() realtimetts.StreamInfo {
    return e.streamInfo
}

func (e *CustomEngine) CanConsumeGenerators() bool {
    return false
}

func (e *CustomEngine) customSynthesis(text string) []byte {
    // 实现具体的音频合成逻辑
    return []byte{}
}
```

## 📊 性能特性

- **低延迟**：首块音频延迟 < 100ms
- **高并发**：支持多个并发TTS请求
- **内存优化**：智能缓冲管理，避免内存溢出
- **设备适配**：自动检测和适配音频设备能力

## 🧪 测试

```bash
# 运行测试
go test ./...

# 运行基准测试
go test -bench=. ./...
```

## 📝 开发指南

### 项目结构

```
realtimetts/
├── types.go          # 核心类型定义
├── engine.go         # 基础引擎实现
├── espeak.go         # ESpeak引擎
├── stream.go         # 文本到音频流处理器
├── client.go         # 客户端API
├── go.mod           # Go模块文件
└── README.md        # 项目说明
```

### 代码规范

- 遵循Go语言官方代码规范
- 使用gofmt格式化代码
- 编写完整的单元测试
- 添加详细的注释和文档

## 🤝 贡献

欢迎贡献代码！请遵循以下步骤：

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🙏 致谢

- [Oto](https://github.com/hajimehoshi/oto) - Go音频播放库
- [ESpeak](http://espeak.sourceforge.net/) - 开源TTS引擎
- Go语言社区

## 📞 联系方式

- 项目主页：https://github.com/yourusername/realtimetts
- 问题反馈：https://github.com/yourusername/realtimetts/issues

---

⭐ 如果这个项目对你有帮助，请给它一个星标！
