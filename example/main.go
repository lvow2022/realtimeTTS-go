package main

import (
	"fmt"
	"log"
	"time"

	"realtimetts"
)

func main() {
	fmt.Println("🎵 RealtimeTTS Go 示例程序")
	fmt.Println("================================")

	// 1. 创建音频配置
	fmt.Println("\n1. 创建音频配置...")
	config := realtimetts.DefaultAudioConfig()
	config.SampleRate = 44100
	config.Channels = 2
	config.Volume = 0.8
	fmt.Printf("   采样率: %d Hz\n", config.SampleRate)
	fmt.Printf("   声道数: %d\n", config.Channels)
	fmt.Printf("   音量: %.1f\n", config.Volume)

	// 2. 创建TTS音频通道
	fmt.Println("\n2. 创建TTS音频通道...")
	ttsChan := make(chan []realtimetts.AudioChunk, 1000)
	fmt.Printf("   TTS通道大小: 1000\n")

	// 3. 创建流播放器
	fmt.Println("\n3. 创建流播放器...")
	player := realtimetts.NewStreamPlayer(ttsChan, config, 1000)

	// 设置回调函数
	player.SetCallbacks(
		// 音频块回调
		func(data []byte) {
			fmt.Printf("   🔊 播放音频块: %d 字节\n", len(data))
		},
		// 单词回调
		func(timing realtimetts.TimingInfo) {
			fmt.Printf("   📝 播放单词: %s (%.2fs - %.2fs)\n",
				timing.Word, timing.StartTime.Seconds(), timing.EndTime.Seconds())
		},
		// 播放开始回调
		func() {
			fmt.Println("   ▶️  播放开始")
		},
		// 播放停止回调
		func() {
			fmt.Println("   ⏹️  播放停止")
		},
		// 播放暂停回调
		func() {
			fmt.Println("   ⏸️  播放暂停")
		},
		// 播放恢复回调
		func() {
			fmt.Println("   ▶️  播放恢复")
		},
	)

	// 5. 模拟音频数据
	fmt.Println("\n5. 模拟音频数据...")

	// 生成多种测试音频数据
	sampleRate := config.SampleRate
	duration := 0.3 // 0.3秒
	numSamples := int(float64(sampleRate) * duration)
	bytesPerSample := config.GetBytesPerFrame()

	// 创建多种音频数据
	audioDataList := make([][]byte, 0)

	// 1. 方波 - 800Hz
	audioData1 := make([]byte, numSamples*bytesPerSample)
	frequency1 := 800.0
	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		amplitude := 0.4
		var sample float32
		if int(t*frequency1)%2 == 0 {
			sample = float32(amplitude)
		} else {
			sample = float32(-amplitude)
		}
		intSample := int16(sample * 32767)
		offset := i * bytesPerSample
		audioData1[offset] = byte(intSample & 0xFF)
		audioData1[offset+1] = byte((intSample >> 8) & 0xFF)
	}
	audioDataList = append(audioDataList, audioData1)

	// 2. 正弦波 - 1200Hz
	audioData2 := make([]byte, numSamples*bytesPerSample)
	frequency2 := 1200.0
	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		amplitude := 0.3
		sample := float32(amplitude * sin(2*3.14159*frequency2*t))
		intSample := int16(sample * 32767)
		offset := i * bytesPerSample
		audioData2[offset] = byte(intSample & 0xFF)
		audioData2[offset+1] = byte((intSample >> 8) & 0xFF)
	}
	audioDataList = append(audioDataList, audioData2)

	// 3. 锯齿波 - 600Hz
	audioData3 := make([]byte, numSamples*bytesPerSample)
	frequency3 := 600.0
	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		amplitude := 0.35
		phase := (t * frequency3) - float64(int(t*frequency3))
		sample := float32(amplitude * (2*phase - 1))
		intSample := int16(sample * 32767)
		offset := i * bytesPerSample
		audioData3[offset] = byte(intSample & 0xFF)
		audioData3[offset+1] = byte((intSample >> 8) & 0xFF)
	}
	audioDataList = append(audioDataList, audioData3)

	// 4. 白噪声
	audioData4 := make([]byte, numSamples*bytesPerSample)
	for i := 0; i < numSamples; i++ {
		amplitude := 0.2
		sample := float32(amplitude * (float64(i%1000)/500.0 - 1.0))
		intSample := int16(sample * 32767)
		offset := i * bytesPerSample
		audioData4[offset] = byte(intSample & 0xFF)
		audioData4[offset+1] = byte((intSample >> 8) & 0xFF)
	}
	audioDataList = append(audioDataList, audioData4)

	// 5. 上升音调
	audioData5 := make([]byte, numSamples*bytesPerSample)
	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		amplitude := 0.3
		frequency := 400.0 + t*800.0 // 从400Hz上升到1200Hz
		sample := float32(amplitude * sin(2*3.14159*frequency*t))
		intSample := int16(sample * 32767)
		offset := i * bytesPerSample
		audioData5[offset] = byte(intSample & 0xFF)
		audioData5[offset+1] = byte((intSample >> 8) & 0xFF)
	}
	audioDataList = append(audioDataList, audioData5)

	// 6. 添加音频数据到TTS通道
	fmt.Println("\n6. 添加音频数据到TTS通道...")
	soundNames := []string{"方波(800Hz)", "正弦波(1200Hz)", "锯齿波(600Hz)", "白噪声", "上升音调"}

	for i := 0; i < 15; i++ { // 增加音频块数量
		audioData := audioDataList[i%len(audioDataList)] // 循环使用不同的音频
		soundName := soundNames[i%len(soundNames)]

		chunk := realtimetts.AudioChunk{
			Data:      audioData,
			Timestamp: time.Now(),
			Duration:  time.Duration(len(audioData)) * time.Second / time.Duration(config.GetBytesPerSecond()),
		}
		select {
		case ttsChan <- []realtimetts.AudioChunk{chunk}:
			fmt.Printf("   添加音频块 %d (%s): %d 字节\n", i+1, soundName, len(audioData))
		default:
			log.Fatalf("TTS通道已满")
		}
	}

	// 7. 显示播放器统计信息
	fmt.Println("\n7. 显示播放器统计信息...")
	playerStats := player.GetStats()
	fmt.Printf("   播放器统计:\n")
	fmt.Printf("     - 已播放字节: %d\n", playerStats.BytesPlayed)
	fmt.Printf("     - 已播放块: %d\n", playerStats.ChunksPlayed)
	fmt.Printf("     - 已播放单词: %d\n", playerStats.WordsPlayed)

	// 8. 开始播放
	fmt.Println("\n8. 开始播放...")
	err := player.Start()
	if err != nil {
		log.Fatalf("开始播放失败: %v", err)
	}
	fmt.Printf("   播放已开始\n")

	// 在播放过程中持续发送音频数据
	fmt.Println("   播放中... (持续发送音频数据)")
	for i := 0; i < 30; i++ { // 发送更多音频块
		audioData := audioDataList[i%len(audioDataList)] // 循环使用不同的音频
		soundName := soundNames[i%len(soundNames)]

		chunk := realtimetts.AudioChunk{
			Data:      audioData,
			Timestamp: time.Now(),
			Duration:  time.Duration(len(audioData)) * time.Second / time.Duration(config.GetBytesPerSecond()),
		}
		select {
		case ttsChan <- []realtimetts.AudioChunk{chunk}:
			fmt.Printf("   发送音频块 %d (%s)\n", i+1, soundName)
		default:
			// 通道满了，等待一下
			time.Sleep(100 * time.Millisecond)
		}
		time.Sleep(150 * time.Millisecond) // 每150ms发送一个块
	}

	// 等待音频播放完成
	time.Sleep(2 * time.Second)

	// 9. 测试播放控制
	fmt.Println("\n9. 测试播放控制...")

	// 设置音量
	err = player.SetVolume(0.7)
	if err != nil {
		log.Fatalf("设置音量失败: %v", err)
	}
	fmt.Printf("   设置音量: %.1f\n", player.GetVolume())

	// 测试静音
	err = player.Mute()
	if err != nil {
		log.Fatalf("设置静音失败: %v", err)
	}
	fmt.Printf("   静音状态: 已静音\n")

	time.Sleep(1 * time.Second)

	err = player.Unmute()
	if err != nil {
		log.Fatalf("取消静音失败: %v", err)
	}
	fmt.Printf("   静音状态: 未静音\n")

	time.Sleep(1 * time.Second)

	// 10. 停止播放
	fmt.Println("\n10. 停止播放...")
	err = player.Stop()
	if err != nil {
		log.Fatalf("停止播放失败: %v", err)
	}
	fmt.Printf("   播放已停止\n")

	// 11. 清理资源
	fmt.Println("\n11. 清理资源...")
	// 关闭TTS通道
	close(ttsChan)
	fmt.Println("   ✅ 资源清理完成")

	fmt.Println("\n🎉 示例程序执行完成！")
	fmt.Println("\n说明:")
	fmt.Println("- 这个示例展示了四个核心模块的基本用法")
	fmt.Println("- 音频流管理器使用PortAudio库进行音频播放")
	fmt.Println("- 缓冲管理器处理音频数据和时间信息")
	fmt.Println("- 流播放器协调整个播放过程")
	fmt.Println("- 所有模块都支持线程安全的操作")
}

// sin 简单的正弦函数实现
func sin(x float64) float64 {
	// 简单的正弦波近似
	x = x - 2*3.14159*float64(int(x/(2*3.14159)))
	if x < 0 {
		x = -x
	}
	if x > 3.14159 {
		x = 2*3.14159 - x
	}
	return x - x*x*x/6 + x*x*x*x*x/120 - x*x*x*x*x*x*x/5040
}
