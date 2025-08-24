package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"realtimetts"
	"realtimetts/engines"
)

func main() {
	// ========================================
	// 交互式TTS演示程序 - 主函数（键盘快捷键版）
	// ========================================
	// 这个程序演示了如何使用RealtimeTTS-Go库创建一个交互式的文本转语音应用
	// 用户可以从控制台输入文本，程序会实时将文本转换为语音并播放
	// 支持键盘快捷键快速控制播放

	fmt.Println("🎵 交互式TTS演示程序（键盘快捷键版）")
	fmt.Println("======================================")

	// 预设测试文本
	presetTexts := map[string]string{
		"1": "你好，这是一个测试文本。",
		"2": "今天天气真不错，适合出去走走。",
		"3": "人工智能技术正在快速发展，改变着我们的生活方式。",
		"4": "这是一个较长的测试文本，用来测试TTS系统的连续播放能力。",
		"5": "语音合成技术让计算机能够像人类一样说话。",
	}

	// ========================================
	// 步骤1: 创建和配置TTS引擎
	// ========================================
	// 使用火山云TTS引擎作为语音合成服务
	// 需要提供AppID、AccessToken和Cluster信息
	fmt.Println("\n1. 创建火山云TTS引擎...")
	volcEngine := engines.NewVolcengineEngine(
		"1882614830",                       // AppID: 火山云应用ID
		"VnqZAkrU_-Ywt3j4zz8D3b5lVeh0U6j5", // AccessToken: 访问令牌
		"volcano_tts",                      // Cluster: 集群名称
	)

	// 配置火山云TTS引擎的详细参数
	volcConfig := engines.VolcengineConfig{
		AppID:         "1882614830",                                  // 应用ID
		AccessToken:   "VnqZAkrU_-Ywt3j4zz8D3b5lVeh0U6j5",            // 访问令牌
		Cluster:       "volcano_tts",                                 // 集群名称
		Endpoint:      "https://openspeech.bytedance.com/api/v1/tts", // API端点
		VoiceType:     "BV700_streaming",                             // 语音类型：流式语音
		Language:      "zh-CN",                                       // 语言：中文
		Rate:          16000,                                         // 采样率：16kHz
		Encoding:      "pcm",                                         // 编码格式：PCM
		SpeedRatio:    1.0,                                           // 语速比例：正常速度
		VolumeRatio:   1.0,                                           // 音量比例：正常音量
		PitchRatio:    1.0,                                           // 音调比例：正常音调
		Channels:      1,                                             // 声道数：单声道
		BitDepth:      16,                                            // 位深度：16位
		FrameDuration: "20ms",                                        // 帧时长：20毫秒
		TextType:      "plain",                                       // 文本类型：纯文本
		Ssml:          false,                                         // 不使用SSML标记
	}

	// 应用配置到引擎
	if err := volcEngine.SetVolcengineConfig(volcConfig); err != nil {
		log.Fatalf("设置火山云配置失败: %v", err)
	}

	// 初始化引擎，建立与火山云服务的连接
	if err := volcEngine.Initialize(); err != nil {
		log.Fatalf("初始化火山云引擎失败: %v", err)
	}

	// 显示引擎信息
	fmt.Printf("   ✅ %s 初始化成功\n", volcEngine.GetEngineInfo().Name)

	// ========================================
	// 步骤2: 配置音频输出参数
	// ========================================
	// 设置音频播放的格式和参数，确保与TTS引擎输出兼容
	fmt.Println("\n2. 配置音频参数...")
	audioConfig := realtimetts.DefaultAudioConfig()
	audioConfig.SampleRate = 16000 // 采样率：16kHz，与火山云TTS保持一致
	audioConfig.Channels = 1       // 声道数：单声道
	audioConfig.BitsPerSample = 16 // 位深度：16位
	audioConfig.Volume = 0.8       // 音量：80%，避免声音过大

	// ========================================
	// 步骤3: 配置流处理参数
	// ========================================
	// 设置文本到音频流的处理参数，控制缓冲和播放行为
	fmt.Println("\n3. 配置TTS流...")
	streamConfig := realtimetts.DefaultStreamConfig()
	streamConfig.AudioConfig = audioConfig    // 使用上面配置的音频参数
	streamConfig.BufferThresholdSeconds = 1.0 // 缓冲阈值：1秒，平衡延迟和连续性
	streamConfig.MinimumSentenceLength = 5    // 最小句子长度：5字符，避免过短文本
	streamConfig.FastSentenceFragment = true  // 快速句子片段：支持不完整句子的快速播放

	// ========================================
	// 步骤4: 设置事件回调函数
	// ========================================
	// 定义各种事件发生时需要执行的回调函数，用于状态监控和用户反馈

	callbacks := realtimetts.NewCallbacks()

	// 播放控制回调：在播放状态变化时通知用户（简洁模式）
	callbacks.OnPlaybackStart = func() {
		fmt.Println("▶️  播放开始")
	}

	callbacks.OnPlaybackStop = func() {
		fmt.Println("⏹️  播放停止")
	}

	callbacks.OnPlaybackPause = func() {
		fmt.Println("⏸️  播放暂停")
	}

	callbacks.OnPlaybackResume = func() {
		fmt.Println("▶️  播放恢复")
	}

	// 系统状态回调：只显示重要错误
	callbacks.OnError = func(err error) {
		fmt.Printf("❌ 错误: %v\n", err)
	}

	// ========================================
	// 步骤5: 创建文本转音频流处理器
	// ========================================
	// 创建主要的TTS流处理器，它将协调文本输入、语音合成和音频播放

	stream := realtimetts.NewTextToAudioStream([]realtimetts.TTSEngine{volcEngine}, streamConfig)
	stream.SetCallbacks(callbacks)

	// ========================================
	// 初始化完成，显示使用说明
	// ========================================
	fmt.Println("\n✅ 初始化完成！")
	fmt.Println("\n📖 使用说明:")
	fmt.Println("  直接输入文本进行TTS播放")
	fmt.Println("  输入 /play 开始播放")
	fmt.Println("  输入 /stop 停止播放")
	fmt.Println("  输入 /pause 暂停播放")
	fmt.Println("  输入 /resume 恢复播放")
	fmt.Println("  输入 /status 查看状态")
	fmt.Println("  输入 /reset 重置TTS状态")
	fmt.Println("  输入 /quit 退出程序")
	fmt.Println("  输入 1-5 快速输入预设文本")
	fmt.Println("")

	// ========================================
	// 步骤6: 启动交互式文本输入处理
	// ========================================
	fmt.Println("🎤 交互式TTS演示已启动，请输入文本或命令:")

	// 创建文本扫描器
	scanner := bufio.NewScanner(os.Stdin)

	// 主循环：处理文本输入
	for {
		fmt.Print("🎵 TTS> ")
		if scanner.Scan() {
			text := strings.TrimSpace(scanner.Text())
			if text == "" {
				continue
			}

			// 处理文本输入
			handleTextInput(text, stream, presetTexts)
		} else {
			// 扫描器出错或EOF
			break
		}
	}
	// ========================================
	// 步骤7: 资源清理和程序退出
	// ========================================
	// 确保在程序退出时正确释放所有资源，避免内存泄漏
	fmt.Println("\n🧹 正在清理资源...")

	// 停止播放：确保音频播放完全停止
	if err := stream.Stop(); err != nil {
		fmt.Printf("⚠️  停止播放失败: %v\n", err)
	}

	// 关闭流：释放TTS流处理器的资源
	if err := stream.Close(); err != nil {
		fmt.Printf("⚠️  关闭流失败: %v\n", err)
	}

	// 关闭引擎：断开与火山云服务的连接
	if err := volcEngine.Close(); err != nil {
		fmt.Printf("⚠️  关闭引擎失败: %v\n", err)
	}

	fmt.Println("✅ 资源清理完成")
	fmt.Println("\n👋 程序已退出，感谢使用！")
}

// handleTextInput 处理文本输入
// 支持特殊命令和普通文本输入
func handleTextInput(text string, stream *realtimetts.TextToAudioStream, presetTexts map[string]string) {
	// 处理特殊命令
	if strings.HasPrefix(text, "/") {
		handleSpecialCommand(text, stream)
		return
	}

	// 处理预设文本快捷输入
	if len(text) == 1 && text >= "1" && text <= "5" {
		if presetText, exists := presetTexts[text]; exists {
			fmt.Printf("📝 预设文本 %s: %s\n", text, presetText)
			text = presetText
		}
	}

	// 处理普通文本输入
	fmt.Printf("📝 输入文本: %s\n", text)

	// 检查并重置异常状态
	status := stream.GetStatus()
	isPlaying := status["is_playing"].(bool)
	if isPlaying {
		fmt.Println("⚠️  检测到异常播放状态，正在重置...")
		if err := stream.Stop(); err != nil {
			fmt.Printf("⚠️  重置状态失败: %v\n", err)
		} else {
			fmt.Println("✅ 状态已重置")
		}
		// 等待一小段时间确保状态完全重置
		time.Sleep(100 * time.Millisecond)
	}

	// 输入文本到流处理器
	if err := stream.Feed(text); err != nil {
		fmt.Printf("❌ 输入文本失败: %v\n", err)
		// 尝试强制重置状态
		fmt.Println("🔄 尝试强制重置状态...")
		if err := stream.Stop(); err != nil {
			fmt.Printf("⚠️  强制重置失败: %v\n", err)
		}
		return
	}

	// 开始播放
	if err := stream.Play(); err != nil {
		fmt.Printf("❌ 开始播放失败: %v\n", err)
		// 播放失败时也要重置状态
		fmt.Println("🔄 播放失败，重置状态...")
		if err := stream.Stop(); err != nil {
			fmt.Printf("⚠️  重置失败: %v\n", err)
		}
		return
	}

	fmt.Println("🎵 开始播放文本...")
}

// handleSpecialCommand 处理特殊命令
func handleSpecialCommand(command string, stream *realtimetts.TextToAudioStream) {
	switch command {
	case "/play":
		// 检查状态并重置异常状态
		status := stream.GetStatus()
		isPlaying := status["is_playing"].(bool)
		if isPlaying {
			fmt.Println("⚠️  检测到异常播放状态，正在重置...")
			if err := stream.Stop(); err != nil {
				fmt.Printf("⚠️  重置状态失败: %v\n", err)
			}
			time.Sleep(100 * time.Millisecond)
		}

		if err := stream.Play(); err != nil {
			fmt.Printf("❌ 播放失败: %v\n", err)
		} else {
			fmt.Println("▶️  开始播放")
		}

	case "/stop":
		if err := stream.Stop(); err != nil {
			fmt.Printf("❌ 停止失败: %v\n", err)
		} else {
			fmt.Println("⏹️  已停止")
		}

	case "/pause":
		if err := stream.Pause(); err != nil {
			fmt.Printf("❌ 暂停失败: %v\n", err)
		} else {
			fmt.Println("⏸️  已暂停")
		}

	case "/resume":
		if err := stream.Resume(); err != nil {
			fmt.Printf("❌ 恢复失败: %v\n", err)
		} else {
			fmt.Println("▶️  已恢复")
		}

	case "/status":
		status := stream.GetStatus()
		fmt.Println("📊 当前状态:")
		fmt.Printf("   播放状态: %v\n", status["is_playing"])
		fmt.Printf("   暂停状态: %v\n", status["is_paused"])
		fmt.Printf("   当前引擎: %v\n", status["current_engine"])
		fmt.Printf("   引擎数量: %v\n", status["engine_count"])

	case "/reset":
		fmt.Println("🔄 正在重置TTS状态...")
		if err := stream.Stop(); err != nil {
			fmt.Printf("⚠️  重置失败: %v\n", err)
		} else {
			fmt.Println("✅ 状态已重置")
		}

	case "/quit":
		fmt.Println("👋 正在退出程序...")
		os.Exit(0)

	default:
		fmt.Printf("❓ 未知命令: %s\n", command)
		fmt.Println("可用命令: /play, /stop, /pause, /resume, /status, /reset, /quit")
	}
}
