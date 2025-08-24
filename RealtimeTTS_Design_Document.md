# RealtimeTTS 系统设计文档

## 目录
1. [系统概述](#系统概述)
2. [架构设计](#架构设计)
3. [核心模块](#核心模块)
4. [数据流设计](#数据流设计)
5. [音频处理架构](#音频处理架构)
6. [线程模型](#线程模型)
7. [回调系统](#回调系统)
8. [错误处理](#错误处理)
9. [性能优化](#性能优化)
10. [扩展性设计](#扩展性设计)
11. [使用指南](#使用指南)
12. [总结](#总结)

## 系统概述

RealtimeTTS 是一个高性能的实时文本转语音系统，采用流式架构设计，支持多种TTS引擎，实现低延迟的语音合成和播放。系统核心特点是边合成边播放，无需等待全部文本处理完成。

### 主要特性
- **实时性**：流式处理，边合成边播放
- **多引擎支持**：Azure、OpenAI、Coqui、ElevenLabs等
- **低延迟**：优化的缓冲策略和异步处理
- **高可靠性**：引擎故障自动切换
- **易扩展**：模块化设计，支持新引擎接入

## 架构设计

### 整体架构图
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Text Input    │───▶│ TextToAudioStream│───▶│   Audio Output  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │   TTSEngine     │◄───┐
                       │   Interface     │    │
                       └─────────────────┘    │
                              │               │
                              ▼               │
                       ┌─────────────────┐    │
                       │   BaseEngine    │────┘
                       │   (Tool Class)  │
                       └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │  StreamPlayer   │
                       └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │  AudioStream    │
                       └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │ PyAudio / mpv   │
                       └─────────────────┘
```

### 重构后的架构特点

**模块化目录结构**：
```
RealtimeTTS-Go/
├── pkg/                    # 核心包
│   ├── types.go           # TTSEngine 接口定义
│   ├── baseEngine.go      # BaseEngine 工具类 (驼峰命名)
│   ├── textToAudioStream.go # 主控制器
│   ├── audioBuffer.go     # 音频缓冲管理
│   ├── streamPlayer.go    # 流播放器
│   ├── audioStream.go     # 音频流管理
│   ├── audioConfig.go     # 音频配置
│   ├── callbacks.go       # 回调系统
│   └── errors.go          # 错误定义
├── engines/               # 引擎实现包
│   ├── volcEngine.go      # 火山云引擎 (驼峰命名)
│   ├── volcEngine_test.go # 火山云引擎测试 (驼峰命名)
│   ├── azureEngine.go     # Azure 引擎实现 (未来)
│   └── openaiEngine.go    # OpenAI 引擎实现 (未来)
├── example/               # 示例程序
│   └── interactive_demo/
└── 其他文件...
```

**包组织设计**：
- **`pkg/` 包**：包含核心功能模块，所有文件属于 `realtimetts` 包
- **`engines/` 包**：包含引擎实现，每个引擎文件属于 `engines` 包
- **模块化设计**：核心功能和引擎实现分离，便于维护和扩展

**依赖注入模式**：
- `TextToAudioStream` 统一创建和管理 `AudioBuffer`
- 通过 `SetAudioBuffer` 方法将 `AudioBuffer` 注入到所有引擎
- 实现了音频缓冲的统一管理和控制

**工具类设计**：
- `BaseEngine` 作为纯工具类，不实现 `TTSEngine` 接口
- 提供基础属性和工具方法，供具体引擎复用
- 具体引擎通过嵌入 `*realtimetts.BaseEngine` 来获得基础功能

**命名规范**：
- 所有Go文件采用驼峰命名法（如 `baseEngine.go`、`volcEngine.go`）
- 测试文件遵循Go测试命名规范（如 `volcEngine_test.go`）
- 包名使用小写字母，符合Go语言规范

### 核心组件
1. **TextToAudioStream**：主控制器，协调整个TTS流程
2. **BaseEngine**：TTS引擎抽象层，定义统一接口
3. **StreamPlayer**：音频播放器，管理播放控制
4. **AudioStream**：音频流管理，处理底层音频接口
5. **AudioBufferManager**：缓冲管理，控制数据流

## 核心模块

### TextToAudioStream 主控制器

**职责**：
- 协调整个TTS流程
- 管理文本输入和音频输出
- 处理播放控制（播放、暂停、停止）
- 管理回调函数和事件处理

**关键属性**：
```python
class TextToAudioStream:
    def __init__(self, engine, ...):
        self.engine = engine          # TTS引擎实例
        self.player = None           # 音频播放器
        self.char_iter = None        # 字符迭代器
        self.play_thread = None      # 播放线程
        self.stream_running = False  # 流状态标志
        self.play_lock = threading.Lock()  # 播放锁
```

**主要方法**：
- `feed()`: 输入文本或迭代器
- `play()`: 同步播放
- `play_async()`: 异步播放
- `pause()/resume()/stop()`: 播放控制
- `load_engine()`: 加载TTS引擎

### BaseEngine 引擎工具类

**设计模式**：组合模式 + 工具类模式

**设计理念**：
- `BaseEngine` 作为纯工具类，不实现 `TTSEngine` 接口
- 提供基础属性和工具方法，供具体引擎复用
- 具体引擎通过嵌入 `*BaseEngine` 来获得基础功能

**核心属性**：
```go
type BaseEngine struct {
    // 基本属性
    engineName           string
    canConsumeGenerators bool
    
    // 音频缓冲管理
    audioBuffer *AudioBuffer
    
    // 回调函数
    onAudioChunk    func([]byte)
    onPlaybackStart func()
    
    // 控制
    stopSynthesisChan chan struct{}
    
    // 音频时长
    audioDuration time.Duration
}
```

**工具方法**：
```go
// 基础工具方法
func (be *BaseEngine) GetEngineName() string
func (be *BaseEngine) SetCanConsumeGenerators(can bool)
func (be *BaseEngine) CanConsumeGenerators() bool

// 回调设置
func (be *BaseEngine) SetOnAudioChunk(callback func([]byte))
func (be *BaseEngine) SetOnPlaybackStart(callback func())

// 控制方法
func (be *BaseEngine) StopSynthesis()
func (be *BaseEngine) ResetAudioDuration()
func (be *BaseEngine) GetAudioDuration() time.Duration

// 默认配置
func (be *BaseEngine) GetDefaultStreamInfo() *AudioConfiguration
func (be *BaseEngine) GetDefaultVoices() []Voice
```

**具体引擎实现示例**：
```go
type AzureEngine struct {
    *BaseEngine  // 嵌入基础功能
    client       *http.Client
    authToken    string
}

func (ae *AzureEngine) GetStreamInfo() *AudioConfiguration {
    // 使用基础工具方法
    config := ae.GetDefaultStreamInfo()
    // 修改为 Azure 特定配置
    config.SampleRate = 24000
    return config
}
```

### StreamPlayer 音频播放器

**架构设计**：
```python
class StreamPlayer:
    def __init__(self, audio_buffer, timings, config, ...):
        self.buffer_manager = AudioBufferManager(audio_buffer, timings, config)
        self.audio_stream = AudioStream(config)
        self.playback_thread = None
        self.playback_active = False
        self.immediate_stop = threading.Event()
        self.pause_event = threading.Event()
```

**核心功能**：
- 音频缓冲管理
- 播放控制（开始、停止、暂停、恢复）
- 格式转换和重采样
- 回调处理和事件通知

## 数据流设计

### 队列机制
```go
// 在BaseEngine中定义
type BaseEngine struct {
    audioBuffer *AudioBuffer    // 音频缓冲管理器
    stopSynthesisChan chan struct{}  // 停止合成通道
}
```

**数据流向**：
1. TTS引擎合成音频 → 通过 `AudioBuffer` 管理
2. `TextToAudioStream` 统一创建和管理 `AudioBuffer`
3. 通过依赖注入将 `AudioBuffer` 注入到所有引擎中
4. 时间信息通过回调函数处理 → 单词级回调

### 缓冲策略
```python
def _synthesis_chunk_generator(self, generator, buffer_threshold_seconds=2.0):
    synthesis_chunk = ""
    for chunk in generator:
        buffered_audio_seconds = self.player.get_buffered_seconds()
        synthesis_chunk += chunk + " "
        
        # 当缓冲音频低于阈值时，产生合成块
        if buffered_audio_seconds < buffer_threshold_seconds:
            yield synthesis_chunk
            synthesis_chunk = ""
```

**缓冲控制参数**：
- `buffer_threshold_seconds`: 控制缓冲时长（秒）
- `minimum_sentence_length`: 最小句子长度（字符）
- `playout_chunk_size`: 播放块大小（字节）
- `frames_per_buffer`: PyAudio缓冲区帧数

## 音频处理架构

### AudioStream 音频流管理
```python
class AudioStream:
    def __init__(self, config: AudioConfiguration):
        self.config = config
        self.stream = None
        self.pyaudio_instance = pyaudio.PyAudio()
        self.actual_sample_rate = 0
        self.mpv_process = None
```

**支持格式**：
- **WAV格式**：使用PyAudio直接播放
- **MPEG格式**：使用mpv播放器处理压缩音频

**设备适配**：
```python
def _get_best_sample_rate(self, device_index, desired_rate):
    # 测试设备支持的采样率
    supported_rates = self.get_supported_sample_rates(device_index)
    
    # 选择最佳采样率
    if desired_rate in supported_rates:
        return desired_rate
    # 选择最接近的采样率
    lower_rates = [r for r in supported_rates if r <= desired_rate]
    if lower_rates:
        return max(lower_rates)
```

### AudioBufferManager 缓冲管理
```python
class AudioBufferManager:
    def __init__(self, audio_buffer, timings, config):
        self.audio_buffer = audio_buffer
        self.timings = timings
        self.total_samples = 0
    
    def get_from_buffer(self, timeout: float = 0.05):
        try:
            chunk = self.audio_buffer.get(timeout=timeout)
            # 计算样本数并更新计数器
            self.total_samples -= len(chunk) // bytes_per_frame
            return True, chunk
        except queue.Empty:
            return False, None
```

**功能**：
- 音频数据缓冲
- 样本计数跟踪
- 缓冲时长计算
- 超时处理

## 线程模型

### 多线程协作
```python
# 主线程：控制播放
self.play_thread = threading.Thread(target=self.play, args=args)

# 工作线程：音频合成
worker_thread = threading.Thread(target=synthesize_worker)

# 播放线程：音频播放
self.playback_thread = threading.Thread(target=self._process_buffer)
```

### 同步机制
```python
# 播放锁：防止并发播放
self.play_lock = threading.Lock()

# 停止事件：立即停止播放
self.immediate_stop = threading.Event()

# 暂停事件：暂停播放
self.pause_event = threading.Event()

# 合成停止事件：停止TTS引擎
self.stop_synthesis_event = mp.Event()
```

### 线程安全
- 使用队列进行线程间通信
- 事件机制控制线程状态
- 锁机制保护共享资源

## 回调系统

### 事件回调
```python
# 文本流事件
on_text_stream_start = None    # 文本流开始
on_text_stream_stop = None     # 文本流结束

# 音频流事件
on_audio_stream_start = None   # 音频流开始
on_audio_stream_stop = None    # 音频流结束

# 处理事件
on_character = None            # 字符级回调
on_word = None                # 单词级回调
on_audio_chunk = None         # 音频块回调
on_sentence_synthesized = None # 句子合成完成回调
```

### 回调时机
- **文本开始**：第一个字符处理时触发
- **音频开始**：第一个音频块播放时触发
- **单词事件**：单词时间信息到达时触发
- **音频块**：每个音频块处理后触发
- **流结束**：文本或音频流结束时触发

### 回调示例
```python
def on_character_callback(char):
    print(f"Processing character: {char}")

def on_word_callback(word):
    print(f"Word spoken: {word}")

def on_audio_chunk_callback(chunk):
    # 处理音频块，如保存到文件
    pass

tts = TextToAudioStream(
    engine=engine,
    on_character=on_character_callback,
    on_word=on_word_callback,
    on_audio_chunk=on_audio_chunk_callback
)
```

## 错误处理

### 引擎故障切换
```python
if not synthesis_successful:
    if len(self.engines) == 1:
        # 只有一个引擎，等待重试
        time.sleep(0.2)
        logging.warning("Engine failed, retrying...")
    else:
        # 切换到下一个引擎
        logging.warning("Switching to fallback engine")
        self.engine_index = (self.engine_index + 1) % len(self.engines)
        self.load_engine(self.engines[self.engine_index])
        self.player.start()
```

### 音频设备错误处理
```python
try:
    self.stream = self.pyaudio_instance.open(
        format=pyFormat,
        channels=pyChannels,
        rate=best_rate,
        output_device_index=pyOutput_device_index,
        frames_per_buffer=self.config.frames_per_buffer,
        output=True,
    )
except Exception as e:
    # 显示可用设备信息
    print("Available Audio Devices:")
    for i in range(device_count):
        device_info = self.pyaudio_instance.get_device_info_by_index(i)
        print(f"Device {i}: {device_info['name']}")
    raise
```

### 异常恢复
- 自动重试机制
- 优雅降级
- 详细错误日志
- 用户友好的错误信息

## 性能优化

### 实时性保证
- **流式处理**：边合成边播放，无需等待全部完成
- **缓冲控制**：动态调整缓冲大小，平衡延迟和连续性
- **异步操作**：合成和播放并行进行，提高效率

### 内存管理
- **队列缓冲**：限制内存使用，避免内存溢出
- **及时清理**：播放完成后清理缓冲数据
- **分块处理**：避免大块数据占用内存

### 延迟优化
- **快速句子片段**：支持不完整句子的快速播放
- **缓冲阈值**：动态调整合成时机
- **设备适配**：自动选择最佳音频参数

### 性能监控
```python
def _on_audio_stream_start(self):
    latency = time.time() - self.stream_start_time
    logging.info(f"Audio stream start, latency to first chunk: {latency:.2f}s")
```

## 扩展性设计

### 引擎扩展
- **统一接口**：所有引擎实现 `TTSEngine` 接口，嵌入 `BaseEngine` 获得基础功能
- **插件化**：可以轻松添加新的TTS引擎，只需实现接口方法
- **配置灵活**：支持引擎特定参数配置
- **依赖注入**：通过 `SetAudioBuffer` 方法实现音频缓冲的依赖注入

### 输出扩展
- **多种格式**：支持WAV、MPEG、MP3等格式
- **多种设备**：支持不同音频设备和输出方式
- **文件输出**：支持保存到文件或流式传输

### 功能扩展
- **多语言支持**：通过tokenizer配置支持不同语言
- **语音参数**：支持音调、速度、音量等调整
- **实时控制**：支持播放过程中的参数调整

### 扩展示例
```go
package engines

import (
    "context"
    "realtimetts/pkg"
)

type CustomEngine struct {
    *pkg.BaseEngine  // 嵌入基础功能
    // 自定义字段
    customConfig map[string]interface{}
}

func NewCustomEngine() *CustomEngine {
    return &CustomEngine{
        BaseEngine: pkg.NewBaseEngine("custom_engine"),
        customConfig: make(map[string]interface{}),
    }
}

func (ce *CustomEngine) GetStreamInfo() *pkg.AudioConfiguration {
    // 使用基础工具方法
    config := ce.GetDefaultStreamInfo()
    // 应用自定义配置
    config.SampleRate = 22050
    return config
}

func (ce *CustomEngine) Synthesize(ctx context.Context, text string) (<-chan []byte, error) {
    // 实现自定义合成逻辑
    outputChan := make(chan []byte, 100)
    go func() {
        defer close(outputChan)
        // 自定义合成实现
        audioData := ce.customSynthesis(text)
        outputChan <- audioData
    }()
    return outputChan, nil
}

func (ce *CustomEngine) GetVoices() ([]pkg.Voice, error) {
    // 返回自定义语音列表
    return []pkg.Voice{
        {ID: "custom_voice_1", Name: "Custom Voice 1", Language: "en"},
    }, nil
}

func (ce *CustomEngine) SetVoice(voice pkg.Voice) error {
    // 自定义语音设置逻辑
    return nil
}

func (ce *CustomEngine) SetVoiceParameters(params map[string]interface{}) error {
    // 自定义参数设置逻辑
    ce.customConfig = params
    return nil
}

func (ce *CustomEngine) SetAudioBuffer(audioBuffer *pkg.AudioBuffer) {
    // 使用基础方法设置音频缓冲
    ce.BaseEngine.SetAudioBuffer(audioBuffer)
}

// 实现其他必需的接口方法
func (ce *CustomEngine) GetEngineInfo() pkg.EngineInfo {
    return pkg.EngineInfo{
        Name:         "Custom Engine",
        Version:      "1.0.0",
        Description:  "Custom TTS Engine",
        Capabilities: []string{"text-to-speech", "voice-selection"},
        Config:       make(map[string]string),
    }
}

func (ce *CustomEngine) Initialize() error {
    return nil
}

func (ce *CustomEngine) Close() error {
    return nil
}
```

## 使用指南

### 基本使用
```go
package main

import (
    "realtimetts/pkg"
    "realtimetts/engines"
)

func main() {
    // 创建引擎
    engine := engines.NewVolcengineEngine("your_app_id", "your_access_token", "your_cluster")
    
    // 创建流
    tts := pkg.NewTextToAudioStream([]pkg.TTSEngine{engine}, nil)
    
    // 输入文本并播放
    tts.Feed("Hello, this is a test.")
    tts.Play()
}
```

### 高级使用
```go
// 异步播放
tts.PlayAsync()

// 播放控制
tts.Pause()
tts.Resume()
tts.Stop()

// 多引擎支持
engineList := []pkg.TTSEngine{
    engines.NewVolcengineEngine("volc_app_id", "volc_token", "volc_cluster"),
    // 未来支持更多引擎
    // engines.NewAzureEngine("azure_key", "region"),
    // engines.NewOpenAIEngine("openai_key"),
}
tts := pkg.NewTextToAudioStream(engineList, nil)

// 设置回调函数
tts.SetCallbacks(&pkg.Callbacks{
    OnAudioChunk: func(data []byte) {
        // 处理音频块
    },
    OnWord: func(word string) {
        // 处理单词事件
    },
})
```

### 配置参数
```go
config := &pkg.StreamConfig{
    AudioConfig:             pkg.DefaultAudioConfig(),
    BufferThresholdSeconds:  2.0,
    MinimumSentenceLength:   10,
    FastSentenceFragment:    true,
    CommaSilenceDuration:    100 * time.Millisecond,
    SentenceSilenceDuration: 300 * time.Millisecond,
    OutputWavFile:           "",
    LogCharacters:           false,
    OutputDeviceIndex:       0,
    Tokenizer:               "nltk",
    Language:                "en",
    Muted:                   false,
}

tts := pkg.NewTextToAudioStream(engineList, config)
```

### 播放参数
```go
// 播放参数通过 StreamConfig 配置
config := &pkg.StreamConfig{
    FastSentenceFragment:    true,           // 快速句子片段
    BufferThresholdSeconds:  2.0,            // 缓冲阈值
    MinimumSentenceLength:   10,             // 最小句子长度
    CommaSilenceDuration:    100 * time.Millisecond,  // 逗号后静音
    SentenceSilenceDuration: 300 * time.Millisecond,  // 句子后静音
    OutputWavFile:           "output.wav",   // 输出文件
}

tts := pkg.NewTextToAudioStream(engineList, config)
```

## 总结

RealtimeTTS系统通过以下设计原则实现了高效的实时语音合成：

### 设计亮点
1. **模块化设计**：清晰的职责分离，易于维护和扩展
2. **流式架构**：边处理边播放，实现真正的实时性
3. **异步处理**：多线程协作，提高系统效率
4. **容错机制**：引擎故障自动切换，提高可靠性
5. **扩展性强**：支持多种引擎和格式，易于集成
6. **实时性好**：优化的缓冲策略，实现低延迟播放
7. **依赖注入**：通过 `SetAudioBuffer` 实现音频缓冲的统一管理
8. **工具类模式**：`BaseEngine` 作为工具类，提供基础功能复用
9. **包组织优化**：`pkg/` 和 `engines/` 分离，便于管理和扩展
10. **命名规范统一**：采用驼峰命名法，符合Go语言最佳实践

### 技术特色
- **队列驱动**：使用队列进行线程间通信，保证数据安全
- **事件驱动**：丰富的回调机制，支持灵活的事件处理
- **设备适配**：自动检测和适配音频设备能力
- **格式兼容**：支持多种音频格式和编码方式
- **性能监控**：内置性能监控和日志记录
- **接口统一**：`TTSEngine` 接口定义统一的引擎行为
- **组合模式**：具体引擎通过嵌入 `pkg.BaseEngine` 获得基础功能
- **包组织**：`pkg/` 核心包和 `engines/` 引擎包分离，便于维护
- **命名规范**：统一的驼峰命名风格，符合Go语言规范
- **Go 语言特性**：充分利用 Go 的并发、接口和组合特性

### 应用场景
- **实时语音助手**：需要低延迟响应的语音交互
- **文本朗读**：长文本的流式朗读
- **语音直播**：实时语音内容生成
- **多语言TTS**：支持多种语言的语音合成
- **音频处理**：音频格式转换和处理

这种设计使得RealtimeTTS既保证了实时性，又具备了良好的可维护性和扩展性，是一个优秀的实时语音合成解决方案。
