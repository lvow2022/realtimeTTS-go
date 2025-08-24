package engines_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"realtimetts/engines"
	realtimetts "realtimetts/pkg"
)

func TestVolcengineEngine(t *testing.T) {
	fmt.Println("🎵 火山云TTS引擎测试程序")
	fmt.Println("================================")

	// 创建测试引擎
	volcEngine := createTestVolcengineEngine(t)
	if volcEngine == nil {
		t.Fatal("创建火山云引擎失败")
	}

	// 运行测试
	testVolcengineEngineBasic(t, volcEngine)
	testVolcengineEngineSynthesis(t, volcEngine)
}

// createTestVolcengineEngine 创建测试用的火山云引擎
func createTestVolcengineEngine(t *testing.T) *engines.VolcengineEngine {
	t.Helper()
	
	fmt.Println("\n1. 创建火山云TTS引擎...")
	volcEngine := engines.NewVolcengineEngine(
		"1882614830",                       // AppID
		"VnqZAkrU_-Ywt3j4zz8D3b5lVeh0U6j5", // AccessToken
		"volcano_tts",                      // Cluster
	)

	// 设置火山云特定配置
	volcConfig := engines.VolcengineConfig{
		AppID:         "1882614830",
		AccessToken:   "VnqZAkrU_-Ywt3j4zz8D3b5lVeh0U6j5",
		Cluster:       "volcano_tts",
		Endpoint:      "https://openspeech.bytedance.com/api/v1/tts",
		VoiceType:     "BV700_streaming",
		Language:      "zh-CN",
		Rate:          16000,
		Encoding:      "pcm",
		SpeedRatio:    1.0,
		VolumeRatio:   1.0,
		PitchRatio:    1.0,
		Channels:      1,
		BitDepth:      16,
		FrameDuration: "20ms",
		TextType:      "plain",
		Ssml:          false,
	}

	if err := volcEngine.SetVolcengineConfig(volcConfig); err != nil {
		t.Fatalf("设置火山云配置失败: %v", err)
	}

	// 初始化引擎
	if err := volcEngine.Initialize(); err != nil {
		t.Fatalf("初始化火山云引擎失败: %v", err)
	}

	fmt.Printf("   引擎名称: %s\n", volcEngine.GetEngineInfo().Name)
	fmt.Printf("   引擎版本: %s\n", volcEngine.GetEngineInfo().Version)
	fmt.Printf("   引擎描述: %s\n", volcEngine.GetEngineInfo().Description)

	return volcEngine
}

// testVolcengineEngineBasic 测试火山云引擎的基本功能
func testVolcengineEngineBasic(t *testing.T, volcEngine *engines.VolcengineEngine) {
	t.Helper()

	// 2. 获取支持的语音列表
	fmt.Println("\n2. 获取支持的语音列表...")
	voices, err := volcEngine.GetSupportedVoices()
	if err != nil {
		t.Logf("获取语音列表失败: %v", err)
	} else {
		for i, voice := range voices {
			fmt.Printf("   语音 %d: %s (%s) - %s\n", i+1, voice.Name, voice.ID, voice.Description)
		}
	}

	// 3. 创建音频配置
	fmt.Println("\n3. 创建音频配置...")
	audioConfig := realtimetts.DefaultAudioConfig()
	audioConfig.SampleRate = 16000 // 改为16000，与火山云TTS保持一致
	audioConfig.Channels = 1
	audioConfig.BitsPerSample = 16
	audioConfig.Volume = 0.8
	fmt.Printf("   采样率: %d Hz (与火山云TTS保持一致)\n", audioConfig.SampleRate)
	fmt.Printf("   声道数: %d\n", audioConfig.Channels)
	fmt.Printf("   位深度: %d\n", audioConfig.BitsPerSample)
	fmt.Printf("   音量: %.1f\n", audioConfig.Volume)

	// 4. 创建流配置
	fmt.Println("\n4. 创建流配置...")
	streamConfig := realtimetts.DefaultStreamConfig()
	streamConfig.AudioConfig = audioConfig
	streamConfig.BufferThresholdSeconds = 1.0
	streamConfig.MinimumSentenceLength = 5
	streamConfig.FastSentenceFragment = true
	fmt.Printf("   缓冲阈值: %.1f秒\n", streamConfig.BufferThresholdSeconds)
	fmt.Printf("   最小句子长度: %d字符\n", streamConfig.MinimumSentenceLength)

	// 5. 创建回调函数
	fmt.Println("\n5. 设置回调函数...")
	callbacks := realtimetts.NewCallbacks()

	// 文本处理回调
	callbacks.OnTextStreamStart = func() {
		fmt.Println("   📝 文本流开始")
	}

	callbacks.OnTextStreamStop = func() {
		fmt.Println("   📝 文本流结束")
	}

	callbacks.OnSentence = func(sentence string) {
		fmt.Printf("   📄 处理句子: %s\n", sentence)
	}

	// 音频处理回调
	callbacks.OnAudioStreamStart = func() {
		fmt.Println("   🔊 音频流开始")
	}

	callbacks.OnAudioStreamStop = func() {
		fmt.Println("   🔊 音频流结束")
	}

	callbacks.OnAudioChunk = func(data []byte) {
		fmt.Printf("   🎵 音频块: %d 字节\n", len(data))
	}

	callbacks.OnSentenceSynthesized = func(sentence string, duration time.Duration) {
		fmt.Printf("   ⏱️  句子合成完成: %s (耗时: %v)\n", sentence, duration)
	}

	// 播放控制回调
	callbacks.OnPlaybackStart = func() {
		fmt.Println("   ▶️  播放开始")
	}

	callbacks.OnPlaybackStop = func() {
		fmt.Println("   ⏹️  播放停止")
	}

	callbacks.OnPlaybackPause = func() {
		fmt.Println("   ⏸️  播放暂停")
	}

	callbacks.OnPlaybackResume = func() {
		fmt.Println("   ▶️  播放恢复")
	}

	// 引擎状态回调
	callbacks.OnEngineReady = func(engineName string) {
		fmt.Printf("   ✅ 引擎就绪: %s\n", engineName)
	}

	callbacks.OnEngineError = func(engineName string, err error) {
		fmt.Printf("   ❌ 引擎错误: %s - %v\n", engineName, err)
	}

	callbacks.OnEngineSwitch = func(oldEngine, newEngine string) {
		fmt.Printf("   🔄 引擎切换: %s -> %s\n", oldEngine, newEngine)
	}

	// 系统状态回调
	callbacks.OnError = func(err error) {
		fmt.Printf("   💥 系统错误: %v\n", err)
	}

	// 6. 创建文本转音频流
	fmt.Println("\n6. 创建文本转音频流...")
	stream := realtimetts.NewTextToAudioStream([]realtimetts.TTSEngine{volcEngine}, streamConfig)
	stream.SetCallbacks(callbacks)

	// 清理资源
	defer func() {
		fmt.Println("\n11. 清理资源...")
		if err := stream.Close(); err != nil {
			t.Logf("关闭流失败: %v", err)
		}
		if err := volcEngine.Close(); err != nil {
			t.Logf("关闭引擎失败: %v", err)
		}
		fmt.Println("\n✅ 测试程序执行完成！")
	}()

	// 7. 测试文本合成
	fmt.Println("\n7. 开始文本合成测试...")

	// 测试文本
	testTexts := []string{
		"你好，这是火山云TTS引擎的测试。",
		"欢迎使用RealtimeTTS Go库。",
		"这是一个实时文本转语音的示例程序。",
		"支持流式处理，边合成边播放。",
	}

	// 输入文本
	for i, text := range testTexts {
		fmt.Printf("\n   输入文本 %d: %s\n", i+1, text)
		if err := stream.Feed(text); err != nil {
			t.Logf("输入文本失败: %v", err)
			continue
		}
	}

	// 8. 开始播放
	fmt.Println("\n8. 开始播放...")
	if err := stream.Play(); err != nil {
		t.Fatalf("开始播放失败: %v", err)
	}

	// 等待播放完成
	fmt.Println("\n   等待播放完成...")

	// 先等待一段时间让播放开始，然后再检测播放完成
	fmt.Println("   等待播放开始...")
	time.Sleep(2 * time.Second)

	// 使用智能等待机制，等待播放真正完成
	waitTimeout := 20 * time.Second // 最大等待时间
	if err := stream.WaitForPlaybackComplete(waitTimeout); err != nil {
		fmt.Printf("   ⚠️  等待播放完成时出错: %v\n", err)
	} else {
		fmt.Println("   ✅ 音频播放已完整完成！")
	}

	// 9. 停止播放
	fmt.Println("\n9. 停止播放...")
	if err := stream.Stop(); err != nil {
		t.Logf("停止播放失败: %v", err)
	}

	// 10. 获取状态信息
	fmt.Println("\n10. 获取状态信息...")
	status := stream.GetStatus()
	fmt.Printf("   播放状态: %v\n", status["is_playing"])
	fmt.Printf("   暂停状态: %v\n", status["is_paused"])
	fmt.Printf("   当前引擎: %v\n", status["current_engine"])
	fmt.Printf("   引擎数量: %v\n", status["engine_count"])

	// 10.1 获取音频统计信息
	fmt.Println("\n10.1 音频统计信息对比...")

	// TTS引擎统计信息
	engineBytesSent, engineChunksSent := volcEngine.GetVolcengineStats()
	fmt.Printf("   🚀 TTS引擎统计:\n")
	fmt.Printf("      发送字节数: %d 字节\n", engineBytesSent)
	fmt.Printf("      发送块数: %d 块\n", engineChunksSent)

	// 缓冲管理器统计信息
	bufferStats := stream.GetBufferStats()
	fmt.Printf("   📦 缓冲管理器统计:\n")
	fmt.Printf("      处理字节数: %d 字节\n", bufferStats.BytesProcessed)
	fmt.Printf("      处理块数: %d 块\n", bufferStats.ChunksProcessed)

	// 播放器统计信息
	playbackStats := stream.GetPlaybackStats()
	fmt.Printf("   🎵 播放器统计:\n")
	fmt.Printf("      播放字节数: %d 字节\n", playbackStats.BytesPlayed)
	fmt.Printf("      播放块数: %d 块\n", playbackStats.ChunksPlayed)

	// 对比分析
	fmt.Printf("\n   📊 数据流分析:\n")
	if engineBytesSent == bufferStats.BytesProcessed && bufferStats.BytesProcessed == playbackStats.BytesPlayed {
		fmt.Printf("      ✅ 所有组件数据量一致！\n")
	} else {
		fmt.Printf("      ⚠️  数据量不一致：\n")
		fmt.Printf("         TTS -> 缓冲器: %d -> %d 字节 (差异: %d)\n",
			engineBytesSent, bufferStats.BytesProcessed, engineBytesSent-bufferStats.BytesProcessed)
		fmt.Printf("         缓冲器 -> 播放器: %d -> %d 字节 (差异: %d)\n",
			bufferStats.BytesProcessed, playbackStats.BytesPlayed, bufferStats.BytesProcessed-playbackStats.BytesPlayed)

		if playbackStats.BytesPlayed < engineBytesSent {
			lossPercentage := float64(engineBytesSent-playbackStats.BytesPlayed) / float64(engineBytesSent) * 100
			fmt.Printf("         📉 播放完整度: %.2f%% (丢失 %.2f%%)\n",
				100.0-lossPercentage, lossPercentage)
		}
	}
}

// testVolcengineEngineSynthesis 测试火山云引擎的合成功能
func testVolcengineEngineSynthesis(t *testing.T, volcEngine *engines.VolcengineEngine) {
	t.Helper()
	
	// 这里可以添加更多的合成测试
	fmt.Println("\n测试火山云引擎合成功能...")
	
	// 测试简单的文本合成
	ctx := context.Background()
	outputChan, err := volcEngine.Synthesize(ctx, "测试文本合成功能")
	if err != nil {
		t.Fatalf("文本合成失败: %v", err)
	}
	
	// 读取一些音频数据
	count := 0
	for audioData := range outputChan {
		if count < 5 { // 只读取前5个音频块
			fmt.Printf("收到音频数据: %d 字节\n", len(audioData))
			count++
		} else {
			break
		}
	}
	
	fmt.Printf("成功接收 %d 个音频块\n", count)
}
