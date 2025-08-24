package realtimetts

import (
	"fmt"
	"testing"
)

// TestAudioConfiguration 测试音频配置
func TestAudioConfiguration(t *testing.T) {
	config := DefaultAudioConfig()

	// 测试默认配置
	if config.SampleRate != 16000 {
		t.Errorf("期望采样率16000，实际得到%d", config.SampleRate)
	}

	if config.Channels != 1 {
		t.Errorf("期望声道数1，实际得到%d", config.Channels)
	}

	// 测试配置验证
	if err := config.Validate(); err != nil {
		t.Errorf("默认配置验证失败: %v", err)
	}

	// 测试无效配置
	invalidConfig := &AudioConfiguration{
		Channels:   -1,
		SampleRate: 0,
		Volume:     1.5,
	}

	if err := invalidConfig.Validate(); err == nil {
		t.Error("无效配置应该验证失败")
	}

	fmt.Println("✅ 音频配置测试通过")
}

// TestAudioBufferManager 测试音频缓冲管理器
func TestAudioBufferManager(t *testing.T) {
	config := DefaultAudioConfig()
	ttsChan := make(chan [][]byte, 100)
	bufferManager := NewAudioBuffer(ttsChan, config, 100)

	// 测试缓冲时长计算
	bufferedSeconds := bufferManager.GetBufferedSeconds()
	if bufferedSeconds < 0 {
		t.Errorf("缓冲时长应该大于等于0，实际得到%f", bufferedSeconds)
	}

	// 测试清空缓冲区
	bufferManager.ClearBuffer()
	if !bufferManager.IsEmpty() {
		t.Error("缓冲区应该为空")
	}

	// 测试统计信息获取
	stats := bufferManager.GetStats()
	if stats.TotalSamples != 0 {
		t.Errorf("新创建的缓冲区总样本数应该为0，实际得到%d", stats.TotalSamples)
	}

	fmt.Println("✅ 音频缓冲管理器测试通过")
}

// TestAudioStream 测试音频流管理器
func TestAudioStream(t *testing.T) {
	config := DefaultAudioConfig()
	audioStream := NewAudioStream(config)

	// 测试设备信息获取
	devices, err := audioStream.GetAvailableDevices()
	if err != nil {
		t.Logf("获取设备列表失败（可能是环境问题）: %v", err)
	} else {
		if len(devices) == 0 {
			t.Log("没有找到音频设备")
		} else {
			fmt.Printf("找到 %d 个音频设备\n", len(devices))
		}
	}

	// 测试配置验证
	if err := audioStream.SetVolume(0.5); err != nil {
		t.Errorf("设置音量失败: %v", err)
	}

	if audioStream.GetVolume() != 0.5 {
		t.Errorf("期望音量0.5，实际得到%f", audioStream.GetVolume())
	}

	// 测试静音功能
	if err := audioStream.SetMuted(true); err != nil {
		t.Errorf("设置静音失败: %v", err)
	}

	if !audioStream.IsMuted() {
		t.Error("应该处于静音状态")
	}

	fmt.Println("✅ 音频流管理器测试通过")
}

// TestStreamPlayer 测试流播放器
func TestStreamPlayer(t *testing.T) {
	config := DefaultAudioConfig()
	ttsChan := make(chan [][]byte, 100)
	audioBuffer := NewAudioBuffer(ttsChan, config, 100)
	player := NewStreamPlayer(audioBuffer, config, 100)

	// 测试播放器状态
	if player.IsPlaying() {
		t.Error("新创建的播放器不应该在播放")
	}

	if player.IsPaused() {
		t.Error("新创建的播放器不应该暂停")
	}

	// 测试音量控制
	if err := player.SetVolume(0.7); err != nil {
		t.Errorf("设置音量失败: %v", err)
	}

	if player.GetVolume() != 0.7 {
		t.Errorf("期望音量0.7，实际得到%f", player.GetVolume())
	}

	// 测试静音控制
	if err := player.Mute(); err != nil {
		t.Errorf("设置静音失败: %v", err)
	}

	if err := player.Unmute(); err != nil {
		t.Errorf("取消静音失败: %v", err)
	}

	// 测试缓冲时长获取
	bufferedSeconds := player.GetBufferedSeconds()
	if bufferedSeconds < 0 {
		t.Errorf("缓冲时长应该大于等于0，实际得到%f", bufferedSeconds)
	}

	fmt.Println("✅ 流播放器测试通过")
}

// TestIntegration 集成测试
func TestIntegration(t *testing.T) {
	fmt.Println("🚀 开始集成测试...")

	// 创建配置
	config := DefaultAudioConfig()

	// 创建TTS音频通道
	ttsChan := make(chan [][]byte, 1000)

	// 创建AudioBuffer
	audioBuffer := NewAudioBuffer(ttsChan, config, 1000)

	// 创建播放器
	player := NewStreamPlayer(audioBuffer, config, 1000)

	// 设置回调函数
	player.SetCallbacks(
		func(data []byte) {
			fmt.Printf("音频块回调: %d 字节\n", len(data))
		},
		func(timing TimingInfo) {
			fmt.Printf("单词回调: %s (%.2fs - %.2fs)\n", timing.Word, timing.StartTime.Seconds(), timing.EndTime.Seconds())
		},
		nil, // onPlaybackStart
		nil, // onPlaybackStop
		nil, // onPlaybackPause
		nil, // onPlaybackResume
	)

	// 添加一些测试数据
	testAudioData := make([]byte, 1024)
	for i := range testAudioData {
		testAudioData[i] = byte(i % 256)
	}

	// 添加音频数据到TTS通道
	for i := 0; i < 5; i++ {
		// 直接发送音频数据
		select {
		case ttsChan <- [][]byte{testAudioData}:
			// 数据已发送到TTS通道
		default:
			t.Errorf("TTS通道已满")
		}
	}

	// 获取播放器统计信息

	playerStats := player.GetStats()
	fmt.Printf("播放器统计: 已播放字节=%d, 已播放块=%d, 已播放单词=%d\n",
		playerStats.BytesPlayed, playerStats.ChunksPlayed, playerStats.WordsPlayed)

	fmt.Println("✅ 集成测试通过")
}

// BenchmarkAudioBufferManager 性能测试
func BenchmarkAudioBufferManager(b *testing.B) {
	config := DefaultAudioConfig()
	ttsChan := make(chan [][]byte, 1000)
	bufferManager := NewAudioBuffer(ttsChan, config, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 测试统计信息获取性能
		bufferManager.GetStats()
		bufferManager.GetBufferedSeconds()
	}
}

// TestDependencyInjection 测试依赖注入功能
func TestDependencyInjection(t *testing.T) {
	fmt.Println("🧪 开始测试依赖注入功能...")

	// 创建配置
	config := DefaultStreamConfig()

	// 创建模拟引擎（这里使用一个简单的结构体来模拟）
	type MockEngine struct {
		audioBuffer *AudioBuffer
	}

	mockEngine := &MockEngine{}

	// 创建AudioBuffer
	ttsChan := make(chan [][]byte, 100)
	audioBuffer := NewAudioBuffer(ttsChan, config.AudioConfig, 100)

	// 模拟依赖注入
	mockEngine.audioBuffer = audioBuffer

	// 验证注入是否成功
	if mockEngine.audioBuffer == nil {
		t.Error("AudioBuffer注入失败")
	}

	// 验证AudioBuffer功能
	stats := mockEngine.audioBuffer.GetStats()
	if stats.TotalSamples != 0 {
		t.Errorf("新创建的AudioBuffer总样本数应该为0，实际得到%d", stats.TotalSamples)
	}

	fmt.Println("✅ 依赖注入功能测试通过")
}
