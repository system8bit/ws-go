package ws

import (
	"encoding/gob"
	"fmt"
	"io"
	"os"
)

// SaveState holds a complete snapshot of the emulator state.
type SaveState struct {
	// CPU
	CPU_AX, CPU_BX, CPU_CX, CPU_DX       uint16
	CPU_SI, CPU_DI, CPU_SP, CPU_BP        uint16
	CPU_CS, CPU_DS, CPU_ES, CPU_SS        uint16
	CPU_IP, CPU_Flags                     uint16
	CPU_Halted, CPU_InterruptEnable       bool
	CPU_PendingIRQ                        int
	CPU_TotalCycles                       uint64

	// Memory Bus
	Bus_IRAM          []byte
	Bus_IOPorts       [256]byte
	Bus_ROMLinearBank byte
	Bus_SRAMBank      byte
	Bus_ROM0Bank      byte
	Bus_ROM1Bank      byte

	// PPU
	PPU_Framebuffer      []byte
	PPU_DisplayBuffer    []byte
	PPU_Scanline         int
	PPU_DispCtrl         byte
	PPU_BackColor        byte
	PPU_CurrentLine      byte
	PPU_LineCompare      byte
	PPU_SpriteBase       byte
	PPU_SpriteFirst      byte
	PPU_SpriteCount      byte
	PPU_MapBase          byte
	PPU_SCR1ScrollX      byte
	PPU_SCR1ScrollY      byte
	PPU_SCR2ScrollX      byte
	PPU_SCR2ScrollY      byte
	PPU_SCR2WinX0        byte
	PPU_SCR2WinY0        byte
	PPU_SCR2WinX1        byte
	PPU_SCR2WinY1        byte
	PPU_SprWinX0         byte
	PPU_SprWinY0         byte
	PPU_SprWinX1         byte
	PPU_SprWinY1         byte
	PPU_LCDCtrl          byte
	PPU_LCDIcons         byte
	PPU_LCDVtotal        byte
	PPU_ShadeLUT         [8]byte
	PPU_MonoPalette      [16][4]byte
	PPU_RenderIRAM       []byte
	PPU_RenderMapBase    byte
	PPU_RenderBackColor  byte
	PPU_VBlankFlag       bool
	PPU_LineMatchFlag    bool
	PPU_SpriteTableCache [2][128][4]byte
	PPU_SpriteCountCache [2]int
	PPU_SpriteFrameActive bool

	// APU (simplified — ring buffer reset on load)
	APU_ChannelFreq    [4]uint16
	APU_ChannelEnabled [4]bool
	APU_ChannelCounter [4]int
	APU_ChannelPos     [4]byte
	APU_ChannelOutput  [4]int16
	APU_ChannelOutputL [4]int16
	APU_ChannelOutputR [4]int16
	APU_ChannelVolume  [4]byte
	APU_NoiseCfg       byte
	APU_SoundCtrl      byte
	APU_OutputCtrl     byte
	APU_NoiseLFSR      uint16
	APU_VoiceVolume    byte
	APU_HyperVoice     byte
	APU_HVoiceCtrl     byte
	APU_HVoiceChanCtrl byte
	APU_SweepValue     int8
	APU_SweepTime      byte
	APU_SweepCounter   int
	APU_SweepDivider   int
	APU_WaveTableBase  uint16
	APU_CyclePos       int
	APU_LastBlipMono   int32

	// Timer
	Timer_Control      byte
	Timer_HBlankPreset uint16
	Timer_VBlankPreset uint16
	Timer_HBlankCount  uint16
	Timer_VBlankCount  uint16

	// DMA
	DMA_SrcAddr uint32
	DMA_DstAddr uint16
	DMA_Length  uint16
	DMA_Control byte

	// Sound DMA
	SDMA_Source      uint32
	SDMA_Length      uint32
	SDMA_SourceSaved uint32
	SDMA_LengthSaved uint32
	SDMA_Control     byte
	SDMA_Timer       byte

	// EEPROM
	EEPROM_IData [1024]byte
	EEPROM_GData []byte
	EEPROM_IAddr uint16
	EEPROM_GAddr uint16
	EEPROM_ICmd  byte
	EEPROM_GCmd  byte

	// Serial
	Serial_Control     byte
	Serial_SendBuf     byte
	Serial_RecvBuf     byte
	Serial_SendLatched bool
	Serial_RecvLatched bool

	// RTC (nil if no RTC)
	HasRTC        bool
	RTC_Sec       byte
	RTC_Min       byte
	RTC_Hour      byte
	RTC_Wday      byte
	RTC_Mday      byte
	RTC_Mon       byte
	RTC_Year      byte
	RTC_Command   byte
	RTC_CmdBuf    [7]byte
	RTC_CmdIndex  byte
	RTC_CmdCount  byte
	RTC_ClockCyc  uint32
}

// Snapshot captures the full emulator state into a SaveState.
func (s *System) Snapshot() *SaveState {
	ss := &SaveState{}

	// CPU
	ss.CPU_AX = s.CPU.AX
	ss.CPU_BX = s.CPU.BX
	ss.CPU_CX = s.CPU.CX
	ss.CPU_DX = s.CPU.DX
	ss.CPU_SI = s.CPU.SI
	ss.CPU_DI = s.CPU.DI
	ss.CPU_SP = s.CPU.SP
	ss.CPU_BP = s.CPU.BP
	ss.CPU_CS = s.CPU.CS
	ss.CPU_DS = s.CPU.DS
	ss.CPU_ES = s.CPU.ES
	ss.CPU_SS = s.CPU.SS
	ss.CPU_IP = s.CPU.IP
	ss.CPU_Flags = s.CPU.Flags
	ss.CPU_Halted = s.CPU.Halted
	ss.CPU_InterruptEnable = s.CPU.InterruptEnable
	ss.CPU_PendingIRQ = s.CPU.PendingIRQ
	ss.CPU_TotalCycles = s.CPU.TotalCycles

	// Bus
	ss.Bus_IRAM = make([]byte, len(s.Bus.IRAM))
	copy(ss.Bus_IRAM, s.Bus.IRAM)
	ss.Bus_IOPorts = s.Bus.IOPorts
	ss.Bus_ROMLinearBank = s.Bus.ROMLinearBank
	ss.Bus_SRAMBank = s.Bus.SRAMBank
	ss.Bus_ROM0Bank = s.Bus.ROM0Bank
	ss.Bus_ROM1Bank = s.Bus.ROM1Bank

	// PPU
	p := s.PPU
	ss.PPU_Framebuffer = make([]byte, len(p.Framebuffer))
	copy(ss.PPU_Framebuffer, p.Framebuffer[:])
	ss.PPU_DisplayBuffer = make([]byte, len(p.DisplayBuffer))
	copy(ss.PPU_DisplayBuffer, p.DisplayBuffer[:])
	ss.PPU_Scanline = p.Scanline
	ss.PPU_DispCtrl = p.DispCtrl
	ss.PPU_BackColor = p.BackColor
	ss.PPU_CurrentLine = p.CurrentLine
	ss.PPU_LineCompare = p.LineCompare
	ss.PPU_SpriteBase = p.SpriteBase
	ss.PPU_SpriteFirst = p.SpriteFirst
	ss.PPU_SpriteCount = p.SpriteCount
	ss.PPU_MapBase = p.MapBase
	ss.PPU_SCR1ScrollX = p.SCR1ScrollX
	ss.PPU_SCR1ScrollY = p.SCR1ScrollY
	ss.PPU_SCR2ScrollX = p.SCR2ScrollX
	ss.PPU_SCR2ScrollY = p.SCR2ScrollY
	ss.PPU_SCR2WinX0 = p.SCR2WinX0
	ss.PPU_SCR2WinY0 = p.SCR2WinY0
	ss.PPU_SCR2WinX1 = p.SCR2WinX1
	ss.PPU_SCR2WinY1 = p.SCR2WinY1
	ss.PPU_SprWinX0 = p.SprWinX0
	ss.PPU_SprWinY0 = p.SprWinY0
	ss.PPU_SprWinX1 = p.SprWinX1
	ss.PPU_SprWinY1 = p.SprWinY1
	ss.PPU_LCDCtrl = p.LCDCtrl
	ss.PPU_LCDIcons = p.LCDIcons
	ss.PPU_LCDVtotal = p.LCDVtotal
	ss.PPU_ShadeLUT = p.ShadeLUT
	ss.PPU_MonoPalette = p.MonoPalette
	if p.RenderIRAM != nil {
		ss.PPU_RenderIRAM = make([]byte, len(p.RenderIRAM))
		copy(ss.PPU_RenderIRAM, p.RenderIRAM)
	}
	ss.PPU_RenderMapBase = p.RenderMapBase
	ss.PPU_RenderBackColor = p.RenderBackColor
	ss.PPU_VBlankFlag = p.VBlankFlag
	ss.PPU_LineMatchFlag = p.LineMatchFlag
	ss.PPU_SpriteTableCache = p.SpriteTableCache
	ss.PPU_SpriteCountCache = p.SpriteCountCache
	ss.PPU_SpriteFrameActive = p.SpriteFrameActive

	// APU
	a := s.APU
	for i := 0; i < 4; i++ {
		ss.APU_ChannelFreq[i] = a.Channels[i].Frequency
		ss.APU_ChannelEnabled[i] = a.Channels[i].Enabled
		ss.APU_ChannelCounter[i] = a.Channels[i].Counter
		ss.APU_ChannelPos[i] = a.Channels[i].Position
		ss.APU_ChannelOutput[i] = a.Channels[i].Output
		ss.APU_ChannelOutputL[i] = a.Channels[i].OutputL
		ss.APU_ChannelOutputR[i] = a.Channels[i].OutputR
	}
	ss.APU_ChannelVolume = a.ChannelVolume
	ss.APU_NoiseCfg = a.NoiseCfg
	ss.APU_SoundCtrl = a.SoundCtrl
	ss.APU_OutputCtrl = a.OutputCtrl
	ss.APU_NoiseLFSR = a.NoiseLFSR
	ss.APU_VoiceVolume = a.VoiceVolume
	ss.APU_HyperVoice = a.HyperVoice
	ss.APU_HVoiceCtrl = a.HVoiceCtrl
	ss.APU_HVoiceChanCtrl = a.HVoiceChanCtrl
	ss.APU_SweepValue = a.SweepValue
	ss.APU_SweepTime = a.SweepTime
	ss.APU_SweepCounter = a.SweepCounter
	ss.APU_SweepDivider = a.GetSweepDivider()
	ss.APU_WaveTableBase = a.WaveTableBase
	ss.APU_CyclePos = a.GetCyclePos()
	ss.APU_LastBlipMono = a.GetLastBlipMono()

	// Timer
	ss.Timer_Control = s.Timer.Control
	ss.Timer_HBlankPreset = s.Timer.HBlankPreset
	ss.Timer_VBlankPreset = s.Timer.VBlankPreset
	ss.Timer_HBlankCount = s.Timer.HBlankCount
	ss.Timer_VBlankCount = s.Timer.VBlankCount

	// DMA
	ss.DMA_SrcAddr = s.DMA.SrcAddr
	ss.DMA_DstAddr = s.DMA.DstAddr
	ss.DMA_Length = s.DMA.Length
	ss.DMA_Control = s.DMA.Control

	// Sound DMA
	ss.SDMA_Source = s.SndDMA.Source
	ss.SDMA_Length = s.SndDMA.Length
	ss.SDMA_SourceSaved = s.SndDMA.SourceSaved
	ss.SDMA_LengthSaved = s.SndDMA.LengthSaved
	ss.SDMA_Control = s.SndDMA.Control
	ss.SDMA_Timer = s.SndDMA.Timer

	// EEPROM
	copy(ss.EEPROM_IData[:], s.EEPROM.IData[:])
	if s.EEPROM.GData != nil {
		ss.EEPROM_GData = make([]byte, len(s.EEPROM.GData))
		copy(ss.EEPROM_GData, s.EEPROM.GData)
	}
	ss.EEPROM_IAddr = s.EEPROM.GetIAddr()
	ss.EEPROM_GAddr = s.EEPROM.GetGAddr()
	ss.EEPROM_ICmd = s.EEPROM.iCmd
	ss.EEPROM_GCmd = s.EEPROM.gCmd

	// Serial
	ss.Serial_Control = s.Serial.Control
	ss.Serial_SendBuf = s.Serial.SendBuf
	ss.Serial_RecvBuf = s.Serial.RecvBuf
	ss.Serial_SendLatched = s.Serial.SendLatched
	ss.Serial_RecvLatched = s.Serial.RecvLatched

	// RTC
	if s.RTC != nil {
		ss.HasRTC = true
		ss.RTC_Sec = s.RTC.GetSec()
		ss.RTC_Min = s.RTC.GetMin()
		ss.RTC_Hour = s.RTC.GetHour()
		ss.RTC_Wday = s.RTC.GetWday()
		ss.RTC_Mday = s.RTC.GetMday()
		ss.RTC_Mon = s.RTC.GetMon()
		ss.RTC_Year = s.RTC.GetYear()
		ss.RTC_Command = s.RTC.GetCommand()
		ss.RTC_CmdBuf = s.RTC.GetCommandBuf()
		ss.RTC_CmdIndex = s.RTC.GetCommandIndex()
		ss.RTC_CmdCount = s.RTC.GetCommandCount()
		ss.RTC_ClockCyc = s.RTC.GetClockCycles()
	}

	return ss
}

// Restore loads a SaveState back into the system.
func (s *System) Restore(ss *SaveState) {
	// CPU
	s.CPU.AX = ss.CPU_AX
	s.CPU.BX = ss.CPU_BX
	s.CPU.CX = ss.CPU_CX
	s.CPU.DX = ss.CPU_DX
	s.CPU.SI = ss.CPU_SI
	s.CPU.DI = ss.CPU_DI
	s.CPU.SP = ss.CPU_SP
	s.CPU.BP = ss.CPU_BP
	s.CPU.CS = ss.CPU_CS
	s.CPU.DS = ss.CPU_DS
	s.CPU.ES = ss.CPU_ES
	s.CPU.SS = ss.CPU_SS
	s.CPU.IP = ss.CPU_IP
	s.CPU.Flags = ss.CPU_Flags
	s.CPU.Halted = ss.CPU_Halted
	s.CPU.InterruptEnable = ss.CPU_InterruptEnable
	s.CPU.PendingIRQ = ss.CPU_PendingIRQ
	s.CPU.TotalCycles = ss.CPU_TotalCycles

	// Bus
	copy(s.Bus.IRAM, ss.Bus_IRAM)
	s.Bus.IOPorts = ss.Bus_IOPorts
	s.Bus.ROMLinearBank = ss.Bus_ROMLinearBank
	s.Bus.SRAMBank = ss.Bus_SRAMBank
	s.Bus.ROM0Bank = ss.Bus_ROM0Bank
	s.Bus.ROM1Bank = ss.Bus_ROM1Bank

	// PPU
	p := s.PPU
	copy(p.Framebuffer[:], ss.PPU_Framebuffer)
	copy(p.DisplayBuffer[:], ss.PPU_DisplayBuffer)
	p.Scanline = ss.PPU_Scanline
	p.DispCtrl = ss.PPU_DispCtrl
	p.BackColor = ss.PPU_BackColor
	p.CurrentLine = ss.PPU_CurrentLine
	p.LineCompare = ss.PPU_LineCompare
	p.SpriteBase = ss.PPU_SpriteBase
	p.SpriteFirst = ss.PPU_SpriteFirst
	p.SpriteCount = ss.PPU_SpriteCount
	p.MapBase = ss.PPU_MapBase
	p.SCR1ScrollX = ss.PPU_SCR1ScrollX
	p.SCR1ScrollY = ss.PPU_SCR1ScrollY
	p.SCR2ScrollX = ss.PPU_SCR2ScrollX
	p.SCR2ScrollY = ss.PPU_SCR2ScrollY
	p.SCR2WinX0 = ss.PPU_SCR2WinX0
	p.SCR2WinY0 = ss.PPU_SCR2WinY0
	p.SCR2WinX1 = ss.PPU_SCR2WinX1
	p.SCR2WinY1 = ss.PPU_SCR2WinY1
	p.SprWinX0 = ss.PPU_SprWinX0
	p.SprWinY0 = ss.PPU_SprWinY0
	p.SprWinX1 = ss.PPU_SprWinX1
	p.SprWinY1 = ss.PPU_SprWinY1
	p.LCDCtrl = ss.PPU_LCDCtrl
	p.LCDIcons = ss.PPU_LCDIcons
	p.LCDVtotal = ss.PPU_LCDVtotal
	p.ShadeLUT = ss.PPU_ShadeLUT
	p.MonoPalette = ss.PPU_MonoPalette
	if ss.PPU_RenderIRAM != nil {
		if p.RenderIRAM == nil || len(p.RenderIRAM) != len(ss.PPU_RenderIRAM) {
			p.RenderIRAM = make([]byte, len(ss.PPU_RenderIRAM))
		}
		copy(p.RenderIRAM, ss.PPU_RenderIRAM)
	}
	p.RenderMapBase = ss.PPU_RenderMapBase
	p.RenderBackColor = ss.PPU_RenderBackColor
	p.VBlankFlag = ss.PPU_VBlankFlag
	p.LineMatchFlag = ss.PPU_LineMatchFlag
	p.SpriteTableCache = ss.PPU_SpriteTableCache
	p.SpriteCountCache = ss.PPU_SpriteCountCache
	p.SpriteFrameActive = ss.PPU_SpriteFrameActive

	// APU
	a := s.APU
	for i := 0; i < 4; i++ {
		a.Channels[i].Frequency = ss.APU_ChannelFreq[i]
		a.Channels[i].Enabled = ss.APU_ChannelEnabled[i]
		a.Channels[i].Counter = ss.APU_ChannelCounter[i]
		a.Channels[i].Position = ss.APU_ChannelPos[i]
		a.Channels[i].Output = ss.APU_ChannelOutput[i]
		a.Channels[i].OutputL = ss.APU_ChannelOutputL[i]
		a.Channels[i].OutputR = ss.APU_ChannelOutputR[i]
	}
	a.ChannelVolume = ss.APU_ChannelVolume
	a.NoiseCfg = ss.APU_NoiseCfg
	a.SoundCtrl = ss.APU_SoundCtrl
	a.OutputCtrl = ss.APU_OutputCtrl
	a.NoiseLFSR = ss.APU_NoiseLFSR
	a.VoiceVolume = ss.APU_VoiceVolume
	a.HyperVoice = ss.APU_HyperVoice
	a.HVoiceCtrl = ss.APU_HVoiceCtrl
	a.HVoiceChanCtrl = ss.APU_HVoiceChanCtrl
	a.SweepValue = ss.APU_SweepValue
	a.SweepTime = ss.APU_SweepTime
	a.SweepCounter = ss.APU_SweepCounter
	a.SetSweepDivider(ss.APU_SweepDivider)
	a.WaveTableBase = ss.APU_WaveTableBase
	a.SetCyclePos(ss.APU_CyclePos)
	a.SetLastBlipMono(ss.APU_LastBlipMono)

	// Timer
	s.Timer.Control = ss.Timer_Control
	s.Timer.HBlankPreset = ss.Timer_HBlankPreset
	s.Timer.VBlankPreset = ss.Timer_VBlankPreset
	s.Timer.HBlankCount = ss.Timer_HBlankCount
	s.Timer.VBlankCount = ss.Timer_VBlankCount

	// DMA
	s.DMA.SrcAddr = ss.DMA_SrcAddr
	s.DMA.DstAddr = ss.DMA_DstAddr
	s.DMA.Length = ss.DMA_Length
	s.DMA.Control = ss.DMA_Control

	// Sound DMA
	s.SndDMA.Source = ss.SDMA_Source
	s.SndDMA.Length = ss.SDMA_Length
	s.SndDMA.SourceSaved = ss.SDMA_SourceSaved
	s.SndDMA.LengthSaved = ss.SDMA_LengthSaved
	s.SndDMA.Control = ss.SDMA_Control
	s.SndDMA.Timer = ss.SDMA_Timer

	// EEPROM
	copy(s.EEPROM.IData[:], ss.EEPROM_IData[:])
	if ss.EEPROM_GData != nil {
		if s.EEPROM.GData == nil || len(s.EEPROM.GData) != len(ss.EEPROM_GData) {
			s.EEPROM.GData = make([]byte, len(ss.EEPROM_GData))
		}
		copy(s.EEPROM.GData, ss.EEPROM_GData)
	}
	s.EEPROM.SetIAddr(ss.EEPROM_IAddr)
	s.EEPROM.SetGAddr(ss.EEPROM_GAddr)
	s.EEPROM.SetICmd(ss.EEPROM_ICmd)
	s.EEPROM.SetGCmd(ss.EEPROM_GCmd)

	// Serial
	s.Serial.Control = ss.Serial_Control
	s.Serial.SendBuf = ss.Serial_SendBuf
	s.Serial.RecvBuf = ss.Serial_RecvBuf
	s.Serial.SendLatched = ss.Serial_SendLatched
	s.Serial.RecvLatched = ss.Serial_RecvLatched

	// RTC
	if ss.HasRTC && s.RTC != nil {
		s.RTC.SetSec(ss.RTC_Sec)
		s.RTC.SetMin(ss.RTC_Min)
		s.RTC.SetHour(ss.RTC_Hour)
		s.RTC.SetWday(ss.RTC_Wday)
		s.RTC.SetMday(ss.RTC_Mday)
		s.RTC.SetMon(ss.RTC_Mon)
		s.RTC.SetYear(ss.RTC_Year)
		s.RTC.SetCommand(ss.RTC_Command)
		s.RTC.SetCommandBuf(ss.RTC_CmdBuf)
		s.RTC.SetCommandIndex(ss.RTC_CmdIndex)
		s.RTC.SetCommandCount(ss.RTC_CmdCount)
		s.RTC.SetClockCycles(ss.RTC_ClockCyc)
	}
}

// SaveToFile saves the current state to a file.
func (s *System) SaveToFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("savestate: create %s: %w", path, err)
	}
	defer f.Close()
	return s.SaveToWriter(f)
}

// SaveToWriter encodes the state to a writer.
func (s *System) SaveToWriter(w io.Writer) error {
	ss := s.Snapshot()
	enc := gob.NewEncoder(w)
	if err := enc.Encode(ss); err != nil {
		return fmt.Errorf("savestate: encode: %w", err)
	}
	return nil
}

// LoadFromFile loads state from a file.
func (s *System) LoadFromFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("savestate: open %s: %w", path, err)
	}
	defer f.Close()
	return s.LoadFromReader(f)
}

// LoadFromReader decodes state from a reader.
func (s *System) LoadFromReader(r io.Reader) error {
	ss := &SaveState{}
	dec := gob.NewDecoder(r)
	if err := dec.Decode(ss); err != nil {
		return fmt.Errorf("savestate: decode: %w", err)
	}
	s.Restore(ss)
	return nil
}
