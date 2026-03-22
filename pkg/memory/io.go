package memory

// IORead reads from an I/O port. Bank register ports return their cached
// values directly; all other ports return from IOPorts (after calling the
// read hook if one is registered, which may update the stored value).
func (b *Bus) IORead(port uint8) byte {
	switch port {
	case PortROMLinearBank:
		return b.ROMLinearBank
	case PortSRAMBank:
		return b.SRAMBank
	case PortROM0Bank:
		return b.ROM0Bank
	case PortROM1Bank:
		return b.ROM1Bank
	default:
		if b.IOReadHook != nil {
			val := b.IOReadHook(port)
			b.IOPorts[port] = val
			return val
		}
		return b.IOPorts[port]
	}
}

// IOWrite writes to an I/O port. Bank register ports update their cached
// fields; all other ports store the value in IOPorts and call the write
// hook if one is registered.
func (b *Bus) IOWrite(port uint8, val byte) {
	switch port {
	case PortROMLinearBank:
		b.ROMLinearBank = val
		b.IOPorts[port] = val
	case PortSRAMBank:
		b.SRAMBank = val
		b.IOPorts[port] = val
	case PortROM0Bank:
		b.ROM0Bank = val
		b.IOPorts[port] = val
	case PortROM1Bank:
		b.ROM1Bank = val
		b.IOPorts[port] = val
	default:
		if b.IOWriteHook != nil {
			b.IOWriteHook(port, val)
		} else {
			b.IOPorts[port] = val
		}
	}
}
