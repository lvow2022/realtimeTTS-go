package realtimetts

import (
	"time"
)

// Callbacks 回调函数集合
// 定义所有可能的回调函数类型
type Callbacks struct {
	// 文本处理回调
	OnCharacter       func(rune)   // 字符级回调
	OnWord            func(string) // 单词级回调
	OnSentence        func(string) // 句子级回调
	OnTextStreamStart func()       // 文本流开始
	OnTextStreamStop  func()       // 文本流结束

	// 音频处理回调
	OnAudioChunk          func([]byte)                // 音频块回调
	OnAudioStreamStart    func()                      // 音频流开始
	OnAudioStreamStop     func()                      // 音频流结束
	OnSentenceSynthesized func(string, time.Duration) // 句子合成完成

	// 播放控制回调
	OnPlaybackStart    func()                             // 播放开始
	OnPlaybackStop     func()                             // 播放停止
	OnPlaybackPause    func()                             // 播放暂停
	OnPlaybackResume   func()                             // 播放恢复
	OnPlaybackProgress func(time.Duration, time.Duration) // 播放进度

	// 引擎状态回调
	OnEngineReady          func(string)         // 引擎就绪
	OnEngineError          func(string, error)  // 引擎错误
	OnEngineSwitch         func(string, string) // 引擎切换
	OnEngineSynthesisStart func(string)         // 引擎合成开始
	OnEngineSynthesisStop  func(string)         // 引擎合成结束

	// 系统状态回调
	OnBufferFull     func()              // 缓冲区满
	OnBufferEmpty    func()              // 缓冲区空
	OnLatencyWarning func(time.Duration) // 延迟警告
	OnError          func(error)         // 通用错误
}

// NewCallbacks 创建新的回调集合
func NewCallbacks() *Callbacks {
	return &Callbacks{
		OnCharacter:            nil,
		OnWord:                 nil,
		OnSentence:             nil,
		OnTextStreamStart:      nil,
		OnTextStreamStop:       nil,
		OnAudioChunk:           nil,
		OnAudioStreamStart:     nil,
		OnAudioStreamStop:      nil,
		OnSentenceSynthesized:  nil,
		OnPlaybackStart:        nil,
		OnPlaybackStop:         nil,
		OnPlaybackPause:        nil,
		OnPlaybackResume:       nil,
		OnPlaybackProgress:     nil,
		OnEngineReady:          nil,
		OnEngineError:          nil,
		OnEngineSwitch:         nil,
		OnEngineSynthesisStart: nil,
		OnEngineSynthesisStop:  nil,
		OnBufferFull:           nil,
		OnBufferEmpty:          nil,
		OnLatencyWarning:       nil,
		OnError:                nil,
	}
}

// SetTextCallbacks 设置文本处理回调
func (c *Callbacks) SetTextCallbacks(
	onCharacter func(rune),
	onWord func(string),
	onSentence func(string),
	onTextStreamStart func(),
	onTextStreamStop func(),
) {
	c.OnCharacter = onCharacter
	c.OnWord = onWord
	c.OnSentence = onSentence
	c.OnTextStreamStart = onTextStreamStart
	c.OnTextStreamStop = onTextStreamStop
}

// SetAudioCallbacks 设置音频处理回调
func (c *Callbacks) SetAudioCallbacks(
	onAudioChunk func([]byte),
	onAudioStreamStart func(),
	onAudioStreamStop func(),
	onSentenceSynthesized func(string, time.Duration),
) {
	c.OnAudioChunk = onAudioChunk
	c.OnAudioStreamStart = onAudioStreamStart
	c.OnAudioStreamStop = onAudioStreamStop
	c.OnSentenceSynthesized = onSentenceSynthesized
}

// SetPlaybackCallbacks 设置播放控制回调
func (c *Callbacks) SetPlaybackCallbacks(
	onPlaybackStart func(),
	onPlaybackStop func(),
	onPlaybackPause func(),
	onPlaybackResume func(),
	onPlaybackProgress func(time.Duration, time.Duration),
) {
	c.OnPlaybackStart = onPlaybackStart
	c.OnPlaybackStop = onPlaybackStop
	c.OnPlaybackPause = onPlaybackPause
	c.OnPlaybackResume = onPlaybackResume
	c.OnPlaybackProgress = onPlaybackProgress
}

// SetEngineCallbacks 设置引擎状态回调
func (c *Callbacks) SetEngineCallbacks(
	onEngineReady func(string),
	onEngineError func(string, error),
	onEngineSwitch func(string, string),
	onEngineSynthesisStart func(string),
	onEngineSynthesisStop func(string),
) {
	c.OnEngineReady = onEngineReady
	c.OnEngineError = onEngineError
	c.OnEngineSwitch = onEngineSwitch
	c.OnEngineSynthesisStart = onEngineSynthesisStart
	c.OnEngineSynthesisStop = onEngineSynthesisStop
}

// SetSystemCallbacks 设置系统状态回调
func (c *Callbacks) SetSystemCallbacks(
	onBufferFull func(),
	onBufferEmpty func(),
	onLatencyWarning func(time.Duration),
	onError func(error),
) {
	c.OnBufferFull = onBufferFull
	c.OnBufferEmpty = onBufferEmpty
	c.OnLatencyWarning = onLatencyWarning
	c.OnError = onError
}

// SafeCall 安全调用回调函数
// 如果回调函数为nil，则不会调用
func (c *Callbacks) SafeCall(callback func(), args ...interface{}) {
	if callback != nil {
		callback()
	}
}

// SafeCallWithArgs 安全调用带参数的回调函数
func (c *Callbacks) SafeCallWithArgs(callback interface{}, args ...interface{}) {
	// 这里可以根据需要实现类型安全的回调调用
	// 为了简化，我们使用反射或者类型断言
	switch cb := callback.(type) {
	case func():
		if cb != nil {
			cb()
		}
	case func(string):
		if cb != nil && len(args) > 0 {
			if str, ok := args[0].(string); ok {
				cb(str)
			}
		}
	case func(error):
		if cb != nil && len(args) > 0 {
			if err, ok := args[0].(error); ok {
				cb(err)
			}
		}
	case func([]byte):
		if cb != nil && len(args) > 0 {
			if data, ok := args[0].([]byte); ok {
				cb(data)
			}
		}
	case func(rune):
		if cb != nil && len(args) > 0 {
			if char, ok := args[0].(rune); ok {
				cb(char)
			}
		}
	case func(time.Duration):
		if cb != nil && len(args) > 0 {
			if duration, ok := args[0].(time.Duration); ok {
				cb(duration)
			}
		}
	case func(time.Duration, time.Duration):
		if cb != nil && len(args) > 1 {
			if duration1, ok := args[0].(time.Duration); ok {
				if duration2, ok := args[1].(time.Duration); ok {
					cb(duration1, duration2)
				}
			}
		}
	case func(string, time.Duration):
		if cb != nil && len(args) > 1 {
			if str, ok := args[0].(string); ok {
				if duration, ok := args[1].(time.Duration); ok {
					cb(str, duration)
				}
			}
		}
	case func(string, error):
		if cb != nil && len(args) > 1 {
			if str, ok := args[0].(string); ok {
				if err, ok := args[1].(error); ok {
					cb(str, err)
				}
			}
		}
	case func(string, string):
		if cb != nil && len(args) > 1 {
			if str1, ok := args[0].(string); ok {
				if str2, ok := args[1].(string); ok {
					cb(str1, str2)
				}
			}
		}
	}
}
