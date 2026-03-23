# ws-go

WonderSwan / WonderSwan Color emulator written in Go + [Ebitengine](https://ebitengine.org/).

## Features

- WonderSwan (mono) and WonderSwan Color support
- V30MZ CPU core (NEC 80186-compatible) — verified against [WSCpuTest](https://github.com/FluBBaOfWard/WSCpuTest)
- Scanline-accurate PPU with BG/FG layers, sprites, and windowing
- 4-channel wavetable audio + Voice D/A + HyperVoice via BlipBuf band-limited synthesis
- Save state (F2 / F3)
- Portrait mode (vertical orientation) — toggle with F4
- SRAM and EEPROM save persistence (`.sav` files)
- Real-time clock (RTC) support

## Requirements

- Go 1.21+
- Platform: macOS, Linux, Windows (Ebitengine supported platforms)

## Build & Run

```bash
go build -o ws-go .
./ws-go <rom-file>
```

Example:

```bash
./ws-go game.wsc
```

## Controls

### Game Input (Landscape mode)

| Key | WonderSwan Button |
|-----|-------------------|
| Arrow keys | X pad (Up/Down/Left/Right) |
| W / A / S / D | Y pad (Up/Left/Down/Right) |
| X | A button |
| Z | B button |
| Enter | Start |

### Game Input (Portrait mode — F4 to toggle)

When the screen is rotated 90° counter-clockwise, the d-pad mapping shifts to
match the new physical orientation:

| Key | WonderSwan Button |
|-----|-------------------|
| Arrow keys | X pad (Left/Up/Right/Down) |
| W / A / S / D | Y pad (Left/Up/Right/Down) |
| X | A button |
| Z | B button |
| Enter | Start |

### Emulator Functions

| Key | Function |
|-----|----------|
| F1 | Reset |
| F2 | Save state |
| F3 | Load state |
| F4 | Toggle portrait / landscape mode |

Save states are stored alongside the ROM file with a `.state` extension.
SRAM/EEPROM saves are stored with a `.sav` extension.

## Supported ROM Formats

| Extension | System |
|-----------|--------|
| `.ws` | WonderSwan (mono) |
| `.wsc` | WonderSwan Color |

## License

MIT
