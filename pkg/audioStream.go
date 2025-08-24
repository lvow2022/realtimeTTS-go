package realtimetts

import (
	"fmt"
	"sync"
	"time"

	"github.com/gordonklaus/portaudio"
)

// AudioStream 音频流管理器
// 使用PortAudio库进行音频播放
type AudioStream struct {
	config *AudioConfiguration
	stream *portaudio.Stream

	mu       sync.RWMutex
	isOpen   bool
	isActive bool
	isClosed bool

	actualSampleRate int
	deviceInfo       *DeviceInfo
	lastError        error

	// 音频数据缓冲区
	audioBuffer chan []float32
	bufferSize  int
}

// DeviceInfo 设备信息结构体
type DeviceInfo struct {
	Index       int    // 设备索引
	Name        string // 设备名称
	MaxChannels int    // 最大声道数
	SampleRates []int  // 支持的采样率
	IsDefault   bool   // 是否为默认设备
}

// NewAudioStream 创建新的音频流管理器
func NewAudioStream(config *AudioConfiguration) *AudioStream {
	return &AudioStream{
		config:           config,
		stream:           nil,
		isOpen:           false,
		isActive:         false,
		isClosed:         false,
		actualSampleRate: 0,
		deviceInfo:       nil,
		lastError:        nil,
		audioBuffer:      make(chan []float32, 100), // 100个音频块的缓冲区
		bufferSize:       100,
	}
}

// OpenStream 打开音频流
func (as *AudioStream) OpenStream() error {
	as.mu.Lock()
	defer as.mu.Unlock()

	if as.isOpen {
		return ErrStreamAlreadyOpen
	}

	if err := as.config.Validate(); err != nil {
		as.lastError = err
		return err
	}

	// 初始化 PortAudio
	if err := portaudio.Initialize(); err != nil {
		as.lastError = fmt.Errorf("初始化PortAudio失败: %w", err)
		return as.lastError
	}

	// 获取默认输出设备
	defaultDevice, err := portaudio.DefaultOutputDevice()
	if err != nil {
		as.lastError = fmt.Errorf("获取默认输出设备失败: %w", err)
		return as.lastError
	}

	// 设置设备信息
	as.deviceInfo = &DeviceInfo{
		Index:       0, // 默认设备索引
		Name:        defaultDevice.Name,
		MaxChannels: defaultDevice.MaxOutputChannels,
		SampleRates: []int{8000, 11025, 16000, 22050, 44100, 48000},
		IsDefault:   true,
	}

	// 选择最佳采样率
	as.actualSampleRate = as.selectBestSampleRate(defaultDevice, as.config.SampleRate)
	fmt.Printf("   实际使用采样率: %d Hz\n", as.actualSampleRate)

	// 创建音频流参数
	streamParams := portaudio.StreamParameters{
		Output: portaudio.StreamDeviceParameters{
			Device:   defaultDevice,
			Channels: as.config.Channels,
			Latency:  time.Millisecond * 50,
		},
		SampleRate:      float64(as.actualSampleRate),
		FramesPerBuffer: as.config.FramesPerBuffer,
	}

	// 打开音频流
	stream, err := portaudio.OpenStream(streamParams, as.audioCallback)
	if err != nil {
		as.lastError = fmt.Errorf("打开音频流失败: %w", err)
		return as.lastError
	}

	as.stream = stream
	as.isOpen = true

	return nil
}

// selectBestSampleRate 选择最佳采样率
func (as *AudioStream) selectBestSampleRate(device *portaudio.DeviceInfo, desiredRate int) int {
	// 优先使用指定的采样率
	fmt.Printf("   设备默认采样率: %.0f Hz\n", device.DefaultSampleRate)
	fmt.Printf("   期望采样率: %d Hz\n", desiredRate)

	// 检查设备是否支持指定采样率
	// 这里我们假设设备支持常见的采样率
	supportedRates := []int{8000, 11025, 16000, 22050, 44100, 48000}
	for _, rate := range supportedRates {
		if rate == desiredRate {
			fmt.Printf("   使用指定采样率: %d Hz\n", desiredRate)
			return desiredRate
		}
	}

	// 如果设备不支持指定采样率，使用最接近的采样率
	closestRate := int(device.DefaultSampleRate)
	minDiff := abs(int(device.DefaultSampleRate) - desiredRate)

	for _, rate := range supportedRates {
		diff := abs(rate - desiredRate)
		if diff < minDiff {
			minDiff = diff
			closestRate = rate
		}
	}

	fmt.Printf("   指定采样率不支持，使用最接近的采样率: %d Hz\n", closestRate)
	return closestRate
}

// abs 返回整数的绝对值
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// audioCallback PortAudio 音频回调函数
func (as *AudioStream) audioCallback(out []float32, info portaudio.StreamCallbackTimeInfo, flags portaudio.StreamCallbackFlags) {
	// 从音频缓冲区获取数据
	select {
	case audioData := <-as.audioBuffer:
		// 复制音频数据到输出缓冲区
		copyLen := len(audioData)
		if copyLen > len(out) {
			copyLen = len(out)
		}
		copy(out[:copyLen], audioData[:copyLen])

		// 如果输出缓冲区还有剩余空间，填充静音
		for i := copyLen; i < len(out); i++ {
			out[i] = 0.0
		}
	default:
		// 如果没有音频数据，输出静音
		for i := range out {
			out[i] = 0.0
		}
	}
}

// StartStream 启动音频流
func (as *AudioStream) StartStream() error {
	as.mu.Lock()
	defer as.mu.Unlock()

	if !as.isOpen {
		return ErrStreamNotOpen
	}

	if as.isActive {
		return ErrStreamAlreadyActive
	}

	if err := as.stream.Start(); err != nil {
		as.lastError = fmt.Errorf("启动音频流失败: %w", err)
		return as.lastError
	}

	as.isActive = true
	return nil
}

// StopStream 停止音频流
func (as *AudioStream) StopStream() error {
	as.mu.Lock()
	defer as.mu.Unlock()

	if !as.isActive {
		return ErrStreamNotActive
	}

	if err := as.stream.Stop(); err != nil {
		as.lastError = fmt.Errorf("停止音频流失败: %w", err)
		return as.lastError
	}

	as.isActive = false
	return nil
}

// CloseStream 关闭音频流
func (as *AudioStream) CloseStream() error {
	as.mu.Lock()
	defer as.mu.Unlock()

	if as.isClosed {
		return nil
	}

	if as.isActive {
		as.isActive = false
	}

	if as.stream != nil {
		if err := as.stream.Close(); err != nil {
			as.lastError = fmt.Errorf("关闭音频流失败: %w", err)
		}
		as.stream = nil
	}

	if err := portaudio.Terminate(); err != nil {
		as.lastError = fmt.Errorf("终止PortAudio失败: %w", err)
	}

	as.isOpen = false
	as.isClosed = true

	return as.lastError
}

// IsStreamActive 检查音频流是否激活
func (as *AudioStream) IsStreamActive() bool {
	as.mu.RLock()
	defer as.mu.RUnlock()

	return as.isActive && as.isOpen && !as.isClosed
}

// IsStreamOpen 检查音频流是否打开
func (as *AudioStream) IsStreamOpen() bool {
	as.mu.RLock()
	defer as.mu.RUnlock()

	return as.isOpen && !as.isClosed
}

// WriteAudioData 写入音频数据
func (as *AudioStream) WriteAudioData(data []byte) error {
	as.mu.RLock()
	if !as.isActive || !as.isOpen || as.isClosed {
		as.mu.RUnlock()
		return ErrStreamNotActive
	}
	as.mu.RUnlock()

	// 将字节数据转换为float32格式
	audioData := as.convertBytesToFloat32(data)

	// 将音频数据放入缓冲区，如果缓冲区满了就等待
	select {
	case as.audioBuffer <- audioData:
		return nil
	default:
		// 缓冲区满了，等待一段时间再尝试
		select {
		case as.audioBuffer <- audioData:
			return nil
		case <-time.After(100 * time.Millisecond): // 等待100ms
			fmt.Printf("   ⚠️  音频缓冲区等待超时，丢弃数据\n")
			return ErrBufferFull
		}
	}
}

// GetDeviceInfo 获取设备信息
func (as *AudioStream) GetDeviceInfo() *DeviceInfo {
	as.mu.RLock()
	defer as.mu.RUnlock()

	return as.deviceInfo
}

// GetActualSampleRate 获取实际采样率
func (as *AudioStream) GetActualSampleRate() int {
	as.mu.RLock()
	defer as.mu.RUnlock()

	return as.actualSampleRate
}

// GetLastError 获取最后的错误
func (as *AudioStream) GetLastError() error {
	as.mu.RLock()
	defer as.mu.RUnlock()

	return as.lastError
}

// SetVolume 设置音量
func (as *AudioStream) SetVolume(volume float64) error {
	if volume < 0.0 || volume > 1.0 {
		return ErrInvalidVolume
	}

	as.mu.Lock()
	defer as.mu.Unlock()

	as.config.Volume = volume
	return nil
}

// GetVolume 获取音量
func (as *AudioStream) GetVolume() float64 {
	as.mu.RLock()
	defer as.mu.RUnlock()

	return as.config.Volume
}

// SetMuted 设置静音状态
func (as *AudioStream) SetMuted(muted bool) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	as.config.Muted = muted
	return nil
}

// IsMuted 检查是否静音
func (as *AudioStream) IsMuted() bool {
	as.mu.RLock()
	defer as.mu.RUnlock()

	return as.config.Muted
}

// GetAvailableDevices 获取可用的音频设备
func (as *AudioStream) GetAvailableDevices() ([]DeviceInfo, error) {
	// 如果PortAudio还没有初始化，先初始化
	if !as.isOpen {
		if err := portaudio.Initialize(); err != nil {
			return nil, fmt.Errorf("初始化PortAudio失败: %w", err)
		}
		// 延迟清理，避免影响后续的OpenStream调用
		defer portaudio.Terminate()
	}

	devices, err := portaudio.Devices()
	if err != nil {
		return nil, fmt.Errorf("获取设备列表失败: %w", err)
	}

	var deviceInfos []DeviceInfo
	for i, device := range devices {
		if device.MaxOutputChannels > 0 {
			deviceInfo := DeviceInfo{
				Index:       i,
				Name:        device.Name,
				MaxChannels: device.MaxOutputChannels,
				SampleRates: []int{8000, 11025, 16000, 22050, 44100, 48000},
				IsDefault:   i == 0,
			}
			deviceInfos = append(deviceInfos, deviceInfo)
		}
	}

	return deviceInfos, nil
}

// convertBytesToFloat32 将字节数据转换为float32格式
func (as *AudioStream) convertBytesToFloat32(data []byte) []float32 {
	// 根据位深度转换
	switch as.config.BitsPerSample {
	case 16:
		return as.convertInt16ToFloat32(data)
	case 24:
		return as.convertInt24ToFloat32(data)
	case 32:
		return as.convertInt32ToFloat32(data)
	default:
		return as.convertInt16ToFloat32(data) // 默认使用16位
	}
}

// convertInt16ToFloat32 将16位整数转换为float32
func (as *AudioStream) convertInt16ToFloat32(data []byte) []float32 {
	result := make([]float32, len(data)/2)
	for i := 0; i < len(data); i += 2 {
		value := int16(data[i]) | int16(data[i+1])<<8
		result[i/2] = float32(value) / 32768.0
	}
	return result
}

// convertInt24ToFloat32 将24位整数转换为float32
func (as *AudioStream) convertInt24ToFloat32(data []byte) []float32 {
	result := make([]float32, len(data)/3)
	for i := 0; i < len(data); i += 3 {
		value := int32(data[i]) | int32(data[i+1])<<8 | int32(data[i+2])<<16
		if value&0x800000 != 0 {
			value |= ^0xFFFFFF
		}
		result[i/3] = float32(value) / 8388608.0
	}
	return result
}

// convertInt32ToFloat32 将32位整数转换为float32
func (as *AudioStream) convertInt32ToFloat32(data []byte) []float32 {
	result := make([]float32, len(data)/4)
	for i := 0; i < len(data); i += 4 {
		value := int32(data[i]) | int32(data[i+1])<<8 | int32(data[i+2])<<16 | int32(data[i+3])<<24
		result[i/4] = float32(value) / 2147483648.0
	}
	return result
}
