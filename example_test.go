package realtimetts

import (
	"fmt"
	"testing"
	"time"
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
	bufferManager := NewAudioBufferManager(ttsChan, config, 100)

	// 测试添加音频数据
	testData := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	err := bufferManager.AddToBuffer(testData)
	if err != nil {
		t.Errorf("添加音频数据失败: %v", err)
	}

	// 测试获取音频数据
	data, err := bufferManager.GetFromBuffer(100 * time.Millisecond)
	if err != nil {
		t.Errorf("获取音频数据失败: %v", err)
	}

	if len(data) != len(testData) {
		t.Errorf("期望数据长度%d，实际得到%d", len(testData), len(data))
	}

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
	player := NewStreamPlayer(ttsChan, config, 100)

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

	// 创建播放器
	player := NewStreamPlayer(ttsChan, config, 1000)

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
	bufferManager := NewAudioBufferManager(ttsChan, config, 1000)

	testData := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bufferManager.AddToBuffer(testData)
		bufferManager.GetFromBuffer(1 * time.Millisecond)
	}
}
