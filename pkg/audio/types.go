package audio

import (
	"github.com/xaionaro-go/audio/pkg/audio/types"
)

type PlayerPCM = types.PlayerPCM
type RecorderPCM = types.RecorderPCM
type Stream = types.Stream
type PlayStream = types.PlayStream
type RecordStream = types.RecordStream

type PCMFormat = types.PCMFormat

const (
	PCMFormatUndefined = types.PCMFormatUndefined
	PCMFormatU8        = types.PCMFormatU8
	PCMFormatS16LE     = types.PCMFormatS16LE
	PCMFormatS16BE     = types.PCMFormatS16BE
	PCMFormatFloat32LE = types.PCMFormatFloat32LE
	PCMFormatFloat32BE = types.PCMFormatFloat32BE
	PCMFormatS24LE     = types.PCMFormatS24LE
	PCMFormatS24BE     = types.PCMFormatS24BE
	PCMFormatS32LE     = types.PCMFormatS32LE
	PCMFormatS32BE     = types.PCMFormatS32BE
	PCMFormatFloat64LE = types.PCMFormatFloat64LE
	PCMFormatFloat64BE = types.PCMFormatFloat64BE
	PCMFormatS64LE     = types.PCMFormatS64LE
	PCMFormatS64BE     = types.PCMFormatS64BE
)

type Encoding = types.Encoding
type EncodingPCM = types.EncodingPCM
type SampleRate = types.SampleRate
type Channel = types.Channel
