package cpu

import "testing"

// testBus is a minimal flat-memory bus for CPU unit tests.
type testBus struct {
	mem [0x10000]byte // 64KB flat memory (segment 0)
	io  [256]byte
}

func (b *testBus) Read8(seg, offset uint16) byte {
	addr := (uint32(seg)<<4 + uint32(offset)) & 0xFFFF
	return b.mem[addr]
}

func (b *testBus) Write8(seg, offset uint16, val byte) {
	addr := (uint32(seg)<<4 + uint32(offset)) & 0xFFFF
	b.mem[addr] = val
}

func (b *testBus) Read16(seg, offset uint16) uint16 {
	lo := b.Read8(seg, offset)
	hi := b.Read8(seg, offset+1)
	return uint16(lo) | uint16(hi)<<8
}

func (b *testBus) Write16(seg, offset uint16, val uint16) {
	b.Write8(seg, offset, byte(val))
	b.Write8(seg, offset+1, byte(val>>8))
}

func (b *testBus) IORead(port uint8) byte  { return b.io[port] }
func (b *testBus) IOWrite(port uint8, val byte) { b.io[port] = val }

// newTestCPU creates a CPU wired to a testBus with CS=DS=SS=ES=0, SP=0xFFFE.
func newTestCPU() (*CPU, *testBus) {
	bus := &testBus{}
	c := &CPU{Bus: bus}
	c.CS = 0
	c.DS = 0
	c.SS = 0
	c.ES = 0
	c.IP = 0
	c.SP = 0xFFFE
	c.Flags = 0
	c.segOverride = -1
	c.PendingIRQ = -1
	return c, bus
}

// loadCode writes instruction bytes at CS:IP (address 0x0000).
func loadCode(bus *testBus, code ...byte) {
	for i, b := range code {
		bus.mem[i] = b
	}
}

// assertFlags checks that specific flags match expected values.
func assertFlags(t *testing.T, c *CPU, expected map[uint16]bool) {
	t.Helper()
	names := map[uint16]string{
		FlagCF: "CF", FlagPF: "PF", FlagAF: "AF",
		FlagZF: "ZF", FlagSF: "SF", FlagOF: "OF",
	}
	for flag, want := range expected {
		if got := c.GetFlag(flag); got != want {
			t.Errorf("flag %s: got %v, want %v (Flags=0x%04X)", names[flag], got, want, c.Flags)
		}
	}
}
