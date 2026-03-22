package ws

import (
	"github.com/system8bit/ws-go/pkg/apu"
	"github.com/system8bit/ws-go/pkg/cart"
	"github.com/system8bit/ws-go/pkg/cpu"
	"github.com/system8bit/ws-go/pkg/input"
	"github.com/system8bit/ws-go/pkg/memory"
	"github.com/system8bit/ws-go/pkg/ppu"
)

// System wires together all WonderSwan emulator components and drives the
// main emulation loop.
type System struct {
	CPU      *cpu.CPU
	Bus      *memory.Bus
	PPU      *ppu.PPU
	APU      *apu.APU
	Cart     *cart.Cartridge
	Input    *input.Input
	DMA      *DMA
	SndDMA   *SoundDMA
	Timer    *Timer
	EEPROM   *EEPROM
	RTC      *RTC
	Serial *Serial
}

// New creates a fully wired WonderSwan system from the given cartridge.
func New(cartridge *cart.Cartridge) *System {
	isColor := cartridge.IsColor()

	bus := memory.NewBus(isColor)
	ppuUnit := ppu.New(isColor)
	apuUnit := apu.New()
	inputUnit := input.New()
	cpuUnit := cpu.New(bus)

	// Share IRAM between bus, PPU and APU so they all see the same memory.
	ppuUnit.IRAM = bus.IRAM
	apuUnit.IRAM = bus.IRAM

	// Initial IRAM snapshot so rendering works even before the first MapBase write.
	ppuUnit.SnapshotTiles()

	// Wire cartridge callbacks into the bus.
	bus.CartRead = cartridge.ReadROM
	bus.CartReadSRAM = cartridge.ReadSRAM
	bus.CartWriteSRAM = cartridge.WriteSRAM

	dmaUnit := &DMA{}
	sndDMAUnit := &SoundDMA{}
	timerUnit := &Timer{}
	eepromUnit := NewEEPROM(cartridge.EEPROMData)
	serialUnit := NewSerial()
	var rtcUnit *RTC
	if cartridge.Header.HasRTC() {
		rtcUnit = NewRTC()
	}

	// I/O read hook: dispatch to PPU, APU, Input, DMA, or Timer based on port range.
	bus.IOReadHook = func(port uint8) byte {
		switch {
		case port <= 0x3F || port == 0x60:
			if ppuUnit.HandlesRead(port) {
				return ppuUnit.ReadPort(port)
			}
			return bus.IOPorts[port]
		case port >= 0x40 && port <= 0x48:
			return dmaUnit.ReadPort(port)
		case port >= 0x4A && port <= 0x52:
			return sndDMAUnit.ReadPort(port)
		case port == 0x6A || port == 0x6B:
			return apuUnit.ReadPort(port)
		case port >= 0x80 && port <= 0x99:
			return apuUnit.ReadPort(port)
		case port == 0xA0:
			// Hardware type (Mednafen: wsc ? 0x87 : 0x86)
			if isColor {
				return 0x87
			}
			return 0x86
		case port == 0xA2 || (port >= 0xA4 && port <= 0xAB):
			return timerUnit.ReadPort(port)
		case port >= 0xBA && port <= 0xBE:
			return eepromUnit.ReadPort(port)
		case port == 0xB1 || port == 0xB3:
			return serialUnit.ReadPort(port)
		case port == 0xB5:
			return inputUnit.ReadPort(port)
		case port == 0xB6:
			// IntStatus read: Mednafen returns single-bit mask for the
			// highest-priority pending interrupt, not the raw status.
			status := bus.IOPorts[IOIntStatus] & bus.IOPorts[IOIntEnable]
			for i := 0; i <= 7; i++ {
				if status&(1<<uint(i)) != 0 {
					return 1 << uint(i)
				}
			}
			return 0
		case port >= 0xC4 && port <= 0xC8:
			return eepromUnit.ReadPort(port)
		case (port == 0xCA || port == 0xCB) && rtcUnit != nil:
			return rtcUnit.ReadPort(port)
		default:
			return bus.IOPorts[port]
		}
	}

	// I/O write hook: dispatch to PPU, APU, Input, DMA, or Timer based on port range.
	// The hook is responsible for storing the value into IOPorts (the default
	// store no longer happens when a hook is registered).
	bus.IOWriteHook = func(port uint8, val byte) {
		// IntStatus (0xB6) write: clears the specified bits (Mednafen: IStatus &= ~V).
		if port == IOIntStatus {
			bus.IOPorts[IOIntStatus] &^= val
			return
		}
		// IntAck (0xB4): clear acknowledged bits in IntStatus
		if port == IOIntAck {
			bus.IOPorts[port] = val
			bus.IOPorts[IOIntStatus] &^= val
			return
		}

		// Default: store the value, then dispatch to subsystem
		bus.IOPorts[port] = val

		switch {
		case port <= 0x3F || port == 0x60:
			ppuUnit.WritePort(port, val)
			// Update IOPorts with masked value (PPU applies write masks)
			bus.IOPorts[port] = val & ppuUnit.PortWriteMask(port)
			// When MapBase changes, snapshot IRAM for stable rendering.
			// The ROM swaps buffers when decompression is complete,
			// so this is the moment tile+map data is consistent.
			if port == 0x07 {
				ppuUnit.SnapshotTiles()
			}
		case port >= 0x40 && port <= 0x48:
			dmaUnit.WritePort(port, val)
			if port == 0x48 && val&0x80 != 0 {
				dmaUnit.Execute(bus)
			}
		case port >= 0x4A && port <= 0x52:
			sndDMAUnit.WritePort(port, val)
		case port == 0x6A || port == 0x6B:
			apuUnit.WritePort(port, val)
		case port >= 0x80 && port <= 0x99:
			apuUnit.WritePort(port, val)
		case port == 0xA2 || (port >= 0xA4 && port <= 0xAB):
			timerUnit.WritePort(port, val)
		case port == 0xB1 || port == 0xB3:
			serialUnit.WritePort(port, val)
		case port == 0xB5:
			inputUnit.WritePort(port, val)
		case port >= 0xBA && port <= 0xBE:
			eepromUnit.WritePort(port, val)
		case port >= 0xC4 && port <= 0xC8:
			eepromUnit.WritePort(port, val)
		case (port == 0xCA || port == 0xCB) && rtcUnit != nil:
			rtcUnit.WritePort(port, val)
		}
	}

	return &System{
		CPU:    cpuUnit,
		Bus:    bus,
		PPU:    ppuUnit,
		APU:    apuUnit,
		Cart:   cartridge,
		Input:  inputUnit,
		DMA:    dmaUnit,
		SndDMA: sndDMAUnit,
		Timer:  timerUnit,
		EEPROM: eepromUnit,
		RTC:    rtcUnit,
		Serial: serialUnit,
	}
}

// Reset resets every component to its power-on state.
func (s *System) Reset() {
	s.CPU.Reset()
	s.PPU.Reset()
	s.APU.Reset()
	s.DMA.Reset()
	s.SndDMA.Reset()
	s.Timer.Reset()
	if s.RTC != nil {
		s.RTC.Reset()
	}
	s.Serial.Reset()

	// Clear I/O ports.
	for i := range s.Bus.IOPorts {
		s.Bus.IOPorts[i] = 0
	}
}

// RunFrame executes one complete frame.
// Total lines per frame is dynamic via LCDVtotal (port 0x16), Mednafen-verified.
func (s *System) RunFrame() {
	// Snapshot IRAM at the start of each frame so rendering uses consistent data.
	// For double-buffering games (Bad Apple), MapBase changes also trigger snapshots
	// during the frame, which will override this with the newly-swapped buffer data.
	s.PPU.SnapshotTiles()

	totalLines := s.PPU.TotalLinesForFrame()

	for line := 0; line < totalLines; line++ {
		s.executeScanline(line)
	}

	// Advance RTC by one frame's worth of cycles.
	if s.RTC != nil {
		s.RTC.Clock(uint32(CyclesPerLine * totalLines))
	}

	// Convert band-limited deltas into output samples and write to ring buffer.
	s.APU.EndFrame()

	// Copy completed framebuffer to display buffer so the frontend
	// never sees a partially-rendered frame.
	s.PPU.SwapBuffers()
}

// executeScanline runs one scanline with Mednafen-compliant event timing.
// Mednafen splits 256 cycles into 3 segments: 128 + 96 + 32, with specific
// events at each boundary.
func (s *System) executeScanline(line int) {
	// Update scanline tracking.
	s.PPU.Scanline = line
	s.PPU.CurrentLine = byte(line)
	s.Bus.IOPorts[0x02] = byte(line) // current line register

	// --- Pre-CPU: Render + Sound DMA #1 ---

	// Render visible scanlines BEFORE running CPU so the rendering
	// sees stable IRAM state (not modified by concurrent decompression).
	if line < VisibleLines {
		s.PPU.RenderScanline(line)
	}

	// Serial communication processing (Mednafen: Comm_Process, once per scanline)
	if s.Serial.Process() {
		raiseInterrupt(s, IRQSerialTX)
	}

	// Sound DMA tick #1 (Mednafen: before first CPU segment)
	s.SndDMA.Check(s.Bus, s.APU)

	// Sprite table caching at line 142 (Mednafen: before CPU execution)
	if line == 142 {
		s.PPU.CacheSpriteTable()
	}

	// VBlank events at the first post-visible line (Mednafen: before CPU)
	if line == VisibleLines {
		s.PPU.FlipSpriteFrame()
		raiseInterrupt(s, IRQVBlank)
		if s.Timer.TickVBlank() {
			raiseInterrupt(s, IRQVBlankTimer)
		}
	}

	// HBlank timer ticks every scanline (Mednafen: before CPU segment 1)
	if s.Timer.TickHBlank() {
		raiseInterrupt(s, IRQHBlankTimer)
	}

	// Unconditional interrupt check at scanline start.
	// Required for dispatching pending interrupts when CPU is halted (HLT)
	// or when interrupts were raised between scanlines.
	s.checkAndDispatchInterrupts()

	// --- Segment 1: 128 cycles ---
	s.runCPUCycles(128)

	// --- Between segments 1 and 2: Sound DMA #2 ---
	s.SndDMA.Check(s.Bus, s.APU)

	// --- Segment 2: 96 cycles ---
	s.runCPUCycles(96)

	// --- Between segments 2 and 3: line counter + LineCompare ---
	// Line-match interrupt (IRQ4) after segment 2 (Mednafen timing)
	if byte(line) == s.Bus.IOPorts[0x03] {
		raiseInterrupt(s, IRQLineMatch)
	}

	// --- Segment 3: 32 cycles ---
	s.runCPUCycles(32)

	// Finalize BlipBuf for this scanline: convert accumulated deltas
	// to band-limited output samples and write to ring buffer.
	s.APU.EndScanline(CyclesPerLine)
}

// runCPUCycles executes CPU instructions for the given number of cycles.
func (s *System) runCPUCycles(cycles int) {
	remaining := cycles
	for remaining > 0 {
		if s.CPU.Halted {
			s.checkAndDispatchInterrupts()
			if s.CPU.Halted {
				s.APU.Tick(remaining)
				return
			}
			// Woke from HLT — account for interrupt dispatch cycles
			intCycles := s.CPU.PendingCycles
			s.CPU.PendingCycles = 0
			if intCycles > 0 {
				s.APU.Tick(intCycles)
				remaining -= intCycles
			}
			if remaining <= 0 {
				return
			}
		}

		step := s.CPU.Step()
		if step == 0 {
			step = 1
		}
		s.APU.Tick(step)
		remaining -= step

		if s.CPU.InterruptEnable {
			s.checkAndDispatchInterrupts()
		}
	}
}

// raiseInterrupt sets the interrupt status bit. The actual dispatch happens
// in checkAndDispatchInterrupts, which is called every CPU step.
func raiseInterrupt(s *System, irq uint8) {
	enableMask := s.Bus.IOPorts[IOIntEnable]
	bit := byte(1 << irq)

	if enableMask&bit == 0 {
		return // this interrupt is masked
	}

	// Set the status bit so the game can poll it / CPU can dispatch it.
	s.Bus.IOPorts[IOIntStatus] |= bit
}

// checkAndDispatchInterrupts checks for pending enabled interrupts and
// dispatches the highest-priority one if IF=1.
func (s *System) checkAndDispatchInterrupts() {
	pending := s.Bus.IOPorts[IOIntStatus] & s.Bus.IOPorts[IOIntEnable]
	if pending == 0 {
		return
	}

	// Any pending interrupt wakes the CPU from HLT
	s.CPU.Halted = false

	if !s.CPU.InterruptEnable {
		return
	}

	// Find highest priority interrupt (Mednafen: lowest bit = highest priority)
	for irq := 0; irq <= 7; irq++ {
		if pending&(1<<uint(irq)) != 0 {
			// Clear the status bit
			s.Bus.IOPorts[IOIntStatus] &^= 1 << uint(irq)

			// Compute vector and dispatch
			base := s.Bus.IOPorts[IOIntBase]
			vector := int(base) + irq
			s.CPU.Interrupt(vector)
			return
		}
	}
}
