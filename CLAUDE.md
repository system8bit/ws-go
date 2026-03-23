# ws-go — WonderSwan Emulator in Go

## Project Overview
WonderSwan (mono/Color) emulator using Go + Ebitengine. Bad Apple (WS mono) の映像がクリーンなシルエットとして表示される。ブロックノイズは修正済み（mono mode での bank bit 無視）。商業ゲーム（例: `gunpei.ws`）も対応済み（ROM バンクマッピングを Mednafen 準拠に修正、BIOS スタブ実装）。

## Build & Run
```bash
go build -o ws-go .
./ws-go bad_apple_ws.ws
```
Test ROMs: `bad_apple_ws.ws` (8MB, WS mono), `swandriving.wsc` (64KB, WSC color, ゲームプレイ動作確認済み)
操作: Arrow=X pad, WASD=Y pad, Z=B, X=A, Enter=Start, **F1=リセット, F2=ステートセーブ, F3=ステートロード**

ヘッドレスフレームチェック:
```bash
go build -o /tmp/framecheck ./cmd/framecheck/
/tmp/framecheck
```

## Architecture
```
main.go                  # Entry point, ROM loading, Ebitengine setup
pkg/
  cpu/                   # V30MZ (80186-compatible) CPU core
    cpu.go               # Registers, Step(), Reset()
    decode.go            # ModRM decoding, resolveModRM() caches address
    execute.go           # ~1900 lines, all opcode dispatch
    flags.go             # Flag helpers (arith/logic)
    interrupts.go        # Interrupt() — push flags/CS/IP, load from IVT
  memory/
    bus.go               # 20-bit address bus, IRAM, bank-switched ROM access
    io.go                # IORead/IOWrite — hook runs BEFORE default store
    banking.go           # Bank register constants
  ppu/
    ppu.go               # PPU state, I/O port read/write, SnapshotTiles()
    render.go            # Scanline renderer (SCR1/SCR2, sprites, window)
    palette.go           # Mono palette + shade LUT, color palette
  apu/                   # 4-ch wavetable audio + Voice D/A + HyperVoice
    blip.go              # BlipBuf band-limited synthesis (Blackman sinc)
  cart/
    cartridge.go         # ROM loading, SRAM/EEPROM, .sav persistence
    header.go            # ROM header, save type detection (SRAM vs EEPROM)
  input/                 # 11-button multiplexed input (Ebitengine-free)
  ws/
    system.go            # Component wiring, RunFrame(), interrupt dispatch
    timing.go            # Constants, IRQ bit mapping
    eeprom.go            # Internal EEPROM (1KB) + Game EEPROM (ports 0xBA-0xC8)
    timer.go             # HBlank/VBlank countdown timers
    dma.go               # General-purpose DMA (ROM→IRAM)
    sdma.go              # Sound DMA (ROM→APU port, 4-24kHz)
frontend/
  game.go                # ebiten.Game implementation
  input.go               # Ebitengine key→button mapping
  screen.go              # 3x scaling
  audio.go               # Ebitengine audio bridge
cmd/
  framecheck/main.go     # Headless frame capture for debugging
```

## Key Hardware Facts (Mednafen-verified)

### Map Entry Format (16-bit little-endian)
```
Bits 0-8:   Tile number (9 bits, 0-511)
Bits 9-12:  Palette index (4 bits, 0-15)
Bit 13:     Tile data bank select
Bit 14:     Horizontal flip
Bit 15:     Vertical flip
```
Source: Mednafen gfx.cpp — `palette=(b2>>1)&15`, `b2&0x2000` for bank, `b2&0x4000` hflip, `b2&0x8000` vflip.

### Tile Data Base Addresses
- **Mono 2bpp**: Always base 0x2000 (bank bit ignored in mono mode, Mednafen-verified). 16 bytes/tile.
- **Color 4bpp**: Bank 0 = 0x4000, Bank 1 = 0x8000 (32 bytes/tile)
- Source: Mednafen `DoGfxDecode()` — `address_base = 0x2000` / `0x4000` for mono

### Timer Control Register (port 0xA2)
```
Bit 0: HBlank timer enable
Bit 1: HBlank timer auto-reload (repeat)
Bit 2: VBlank timer enable
Bit 3: VBlank timer auto-reload (repeat)
```

### Shade LUT (ports 0x1C-0x1F)
8 entries, maps shade values 0-7 to LCD gray levels 0-15. Bad Apple sets: `[0,2,5,6,8,10,13,15]`.

### IntStatus (port 0xB6)
- Read: returns **single-bit mask** for highest-priority pending interrupt (`1 << IOn_Which`), Mednafen-verified。NOT the raw status bitmask
- Write: clears specified bits (`IStatus &= ~val`, Mednafen-verified)
- Port 0xB4 (IntAck): same clear behavior

### Interrupt Priority (Mednafen-verified)
- **Bit 0 = highest priority, bit 7 = lowest** (Mednafen: `for(i=0;i<8;i++)`)
- 割り込みディスパッチは 32 サイクル、IRET は 10 サイクル

### Color 0 Transparency (BG layers, Mednafen-verified)
- **SCR1 (BG)**: `if(wsTileRow[x] || !(palette & 0x4))` — 色0は palette bit2=0 なら不透明、bit2=1 なら透過
- **SCR2 (FG) カラーモード** (`wsVMode & 0x2`): `if(wsTileRow[x])` — **色0は常に透過**（palette bit2 無関係）
- **SCR2 (FG) モノモード**: SCR1 と同じルール（palette bit2 で判定）
- swandriving.wsc で検証済み: SCR2 の空タイル(色0)が透過になりSCR1の道路が表示される

### カラーパレットフォーマット (WSC 12-bit RGB)
`xxxx BBBB GGGG RRRR` — bits 0-3 = **Red**, bits 4-7 = **Green**, bits 8-11 = **Blue** (Mednafen-verified)

### IOWrite Hook Order
`bus.IOWrite()` でフックが登録されている場合、**フックが先に実行され、デフォルトストア(`IOPorts[port]=val`)は行われない**。フック内で明示的に `IOPorts[port]=val` する。IntStatus (0xB6) はフックが `&^= val` のみ行い、値を直接格納しない。

## IRAM Snapshot System
- **タイミング**: MapBase (port 0x07) 変更時に `PPU.SnapshotTiles()` を呼ぶ
- **対象**: IRAM 全体 + MapBase + BackColor をコピー
- **目的**: ROM のバッファスワップ時点のデータを保存。デコンプレッション中のタイルデータ変更がレンダリングに影響しない
- **レンダリング**: `render.go` の全関数が `p.RenderIRAM` / `p.RenderMapBase` / `p.RenderBackColor` を使用

## Bad Apple ROM の動作

### メモリレイアウト (IRAM 16KB mono)
```
0x0000-0x07FF: IVT + workspace
0x0800-0x0FFF: SCR2 map buffer 1 (MapBase=0x10)
0x1000-0x17FF: SCR2 map buffer 2 (MapBase=0x20)
0x1800-0x1FFF: workspace
0x2000-0x3FFF: Tile data (512 tiles × 16 bytes)
```

### ISR
- **VBlank (FF9C:002D)**: フレームカウンタ [004A] をインクリメント + ACK のみ。デコンプレッションは行わない。
- **HBlank (FF9C:0039)**: ROM1Bank (segment 0x3000) から APU ポート 0x89 にサウンドデータを送信。毎スキャンライン発火。

### メインループ (FFAA:0109-)
1. HLT で VBlank 待ち（[004A] >= [0050] まで）
2. MapBase 書き込み（バッファスワップ）
3. DispCtrl+BackColor を [0048] から OUT 0x00
4. memcpy: active → inactive (FF9C:0092, 1136 bytes)
5. デルタデコンプレッション (FFCF:0000): ROM から読み、inactive buffer + tile data に書き込み
6. 1に戻る

### デコンプレッション速度
- 2-3 フレーム/サイクル（平均 2.5）= 映像は約 30fps
- デコンプレッションは VBlank から開始し、次フレームの可視ラインにまたがる

### タイル0 の特殊用途
Init コードが IRAM[0x2000-0x200F] を 0xFF で埋める（全ピクセル色3）。
- entry 0x0000 (T0/P0): 全色3 → palette 0 shade 7 = **黒塗り**
- entry 0x0200 (T0/P1): 全色3 → palette 1 shade 0 = **白塗り**

## 既知の問題と次のステップ

### 1. ブロックノイズ（修正済み）
**原因**: `getTilePixel()` のモノラルモードで bank bit を尊重していた。Mednafen の `wsGetTile()` は mono mode (wsVMode==0) で bank bit を無視し常に base 0x2000 を使用。bank=1 だとアドレスが 0x0000 (IVT 領域) にラップし、ゴミデータをタイルとして読んでいた。

**修正内容**:
- `getTilePixel()`: mono mode で bank 無視、常に base 0x2000
- `renderBGLayerWindowed()`: map entry のビットフィールド解析が完全に誤っていた（palette/flip/bank の位置）を修正
- `renderBGLayerWindowed()`: `p.MapBase` → `p.RenderMapBase` に修正
- BG レイヤーに色0透過チェック追加 (`palette & 4` で色0を透過)

### 1b. スプライトレンダリング（修正済み）
**修正内容**:
- スプライトエントリのバイト順修正: tile, attr, Y, X (旧: Y, X, tile, attr)
- 属性ビット位置修正: priority=bit5, hflip=bit6, vflip=bit7 (旧: bit3, bit5, bit6)
- スプライト優先度実装: priority bit OR FGDrawn[] で SCR2 との前後関係を処理
- スプライトウィンドウ (DispCtrl bit3) 実装
- ダブルバッファリング: ライン 142 でスプライトテーブルをキャッシュ、VBlank でバッファ切替
- Y 座標の符号付き処理 (Y>150 → negative offset)
- X 座標の負値対応 (X>=249 → X-=256)
- 色0透過にパレット bit2 ルールを適用

### 1c. フレームバッファ ダブルバッファリング（修正済み）
リアルタイム描画時に白モザイクが出る問題を解消。`Framebuffer` (レンダリング用) → `DisplayBuffer` (表示用) を `RunFrame()` 完了後にコピー。`frontend/game.go` の `Draw()` は `DisplayBuffer` を読む。

### 2. WSC カラーモード対応（基本実装済み）
- `swandriving.wsc` でタイトル画面のカラー表示を確認済み
- **ROM ヘッダパーシング修正**: 最後 10 バイト（旧: 16 バイト）。MinimumSystem が正しく読めていなかった
- **port 0xA0**: Mednafen 準拠で WSC=0x87, WS=0x86 を返す（旧: 0x02/0x00）
- **wsVMode**: `LCDCtrl (port 0x60) >> 5` で動的にビデオモード決定。0=mono 2bpp, 6=color planar 4bpp, 7=color packed 4bpp
- **Mode 7 (packed 4bpp)**: Mednafen 準拠。4 bytes/row, 各バイトが 2 ピクセル（高ニブル=左, 低ニブル=右）
- **Mode 6 (planar 4bpp)**: 4 bitplanes/row。既存実装を維持
- **カラーパレット**: IRAM 0xFE00-0xFFFF から直接読み取り (16 palettes × 16 colors × 2 bytes = 512 bytes, 12-bit RGB)。ColorPalette キャッシュを廃止
- **IRAM スナップショット**: 毎フレーム開始時に IRAM 全体をスナップショット。MapBase 変更時にも追加スナップショット（Bad Apple 互換）
- **renderBackground**: カラーモードでは colIdx を 4bit (0-15) で使用（mono は 2bit マスク維持）
- **swandriving.wsc 動作確認**: 道路・草地・車スプライト正常表示。矢印キーで操作可能
- **衝突時**: ISR内の `|posY-tgtY|<6 && |posX-tgtX|<6` → `EB FE` (JMP $) 無限ループ。デモROMにリスタート実装なし。F1でリセット
- **カラーパレット R/B スワップ修正**: 旧実装は bits 0-3=Blue, 8-11=Red (逆)。正しくは bits 0-3=Red, 8-11=Blue
- **SCR2 color 0 透過修正**: カラーモード(wsVMode&2)で SCR2 の色0を常に透過に変更（Mednafen gfx.cpp 検証済み）

### 3. サウンド DMA（実装済み）
- Sound DMA (ports 0x4A-0x52) 実装済み（`pkg/ws/sdma.go`）
- ROM → APU ポート (0x89 or 0x95) への自動転送
- 4段階レート: 4kHz/6kHz/12kHz/24kHz（タイマー分周）
- 自動リロード（ループ）対応
- スキャンラインあたり 2 回チェック（Mednafen 準拠）

### 3b. APU ポートマッピング修正（実装済み）
- Port 0x88-0x8B: per-channel volume (L/R nibbles)（旧: MasterVolume/SoundOutput）
- Port 0x94: VoiceVolume（Voice D/A モードの L/R ルーティング）
- Port 0x95: HyperVoice、Port 0x6A/0x6B: HyperVoice 制御
- Voice/D/A モード (SoundCtrl bit5): CH2 が volume[1] を直接 PCM サンプルとして出力
- Bad Apple は HBlank ISR で port 0x89 に PCM データを毎スキャンライン送信 → Voice D/A モードで音声出力
- ステレオミキシング対応（OutputL/OutputR per channel）

### 4. CPU 命令精度（WSCpuTest 全標準テスト PASS + 未定義命令 29/34 PASS）
- ADC/SBB のフラグ計算を修正済み（XOR ベースの AF 計算）
- SHL/SHR/SAR の count >= operand_size ガード追加済み
- **POPF/IRET**: bits 3,5 を `&^ 0x0028` でマスク（V30MZ hardwired-0）
- **NEG**: OF=(val==0x80), AF=(val&0x0F!=0)。全算術フラグ設定
- **ROL/ROR/RCL/RCR**: 全 count で OF 設定（count=0 でも CF^MSB 計算）
- **SHL**: OF=CF^result_MSB（全 count）、AF=0、count=0 でも SZP 設定・CF 保持
- **SHR**: OF=result_MSB^result_(MSB-1)、AF=0、count=0 でも SZP 設定・CF 保持
- **SAR**: OF=result_MSB^result_(MSB-1)、AF=0、count=0 でも SZP 設定・CF 保持
- **TF (Trap Flag)**: INT 1 シングルステップトラップ実装（tfBefore で命令前の TF を保存）
- **DAA/DAS**: original AL で高ニブル判定 (>0x99)、CF/AF は条件ベース、OF=signed_overflow(adjusted-original)
- **AAA/AAS**: AL 先マスク (& 0x0F)、V30MZ 固定フラグ (adj: CF=AF=ZF=1,SF=0; no adj: SF=1,ZF=0; 共通: PF=1,OF=0)
- **IMUL 1-operand**: Mednafen 準拠で CF=OF=(high!=0)（符号拡張チェックから変更）
- **ボタンマッピング修正**: A=bit2(0x04), B=bit3(0x08), Start=bit1(0x02)
- **LEA/LES/LDS mod=3**: V30MZ 未定義動作。base_for_rm + register_value(rm) でアドレス計算
- **CLI/STI**: 4 サイクル（旧: 1）
- **タイマープリセット書き込み**: プリセット (0xA4-0xA7) 書き込み時にカウンタも即時ロード (Mednafen 準拠)
- **PPU I/O 書き込みマスク**: DispCtrl=0x3F, SpriteBase=0x1F/0x3F, MapBase=0x77, MonoPalette=0x77/0x70 等
- **PUSH SP**: V30/8086 動作（デクリメント後の値をプッシュ）
- **PUSHF**: ハードワイヤードビット 1,12-15 を常にセット (0xF002)
- **SAHF**: マスク 0xD5 適用（CF,PF,AF,ZF,SF のみ変更）
- **AAM/AAD**: Mednafen 準拠で即値無視・常に 10 使用、AAM は SZP を AX ワード基準で設定
- **NOP**: 3 サイクル（Mednafen 準拠）
- **LOCK prefix (0xF0)**: NOP として処理
- **割り込みディスパッチ**: 32 サイクル（Mednafen: `v30mz_int` CLK(32)）。`PendingCycles` で次の `Step()` に加算
- **IRET**: 10 サイクル（旧: 5）。Mednafen 準拠
- **INT 3/INT imm8**: 10 サイクル + Interrupt() の 32 = 計 42（旧: 5）
- **HLT**: halted 状態ではスキャンライン残りサイクルを一括消費（旧: 1 サイクルずつループ）
- **割り込み優先度**: bit 0 = 最高優先、bit 7 = 最低優先（Mednafen: `for(i=0;i<8;i++)`）。旧実装は逆だった
- **port 0xB6 読み取り**: 最優先ペンディング割り込みの単一ビットマスクを返す（旧: raw IStatus）

### 5a. RTC（実装済み）
- **ポート 0xCA-0xCB**: Mednafen rtc.cpp 準拠。コマンド (0x15=読取, 0x14=設定, 0x13=ACK) + データレジスタ
- **BCD 時計**: 秒→年カスケード、うるう年判定 (Mednafen GenericRTC::Clock 準拠)
- **初期化**: システムローカル時刻から BCD 変換。`cartridge.Header.HasRTC()` が true の場合のみ生成
- **クロック**: `RunFrame()` 末尾で `CyclesPerFrame` サイクル加算、3,072,000 サイクル (1秒) ごとに tick
- **RTC 使用ゲーム**: Dicing Knight, Dokodemo Hamster 3, Inuyasha

### 5b. BOUND 命令 (0x62)（実装済み）
- Mednafen `i_chkind` 準拠。ModRM から下限・上限ワードを読み、レジスタ値が範囲外なら INT 5
- 13 サイクル（Mednafen CLK(13)）

### 5c. REP 割り込みインターリーブ（実装済み）
- REP 文字列命令を1イテレーションずつ実行し、CX > 0 なら IP を REP プレフィックスに巻き戻し
- `instrStartIP` に命令開始アドレスを保存、巻き戻しに使用
- システムループが各イテレーション間で割り込みチェック可能に
- Mednafen の CHK_ICOUNT マクロと同等の効果（PC 巻き戻し方式）
- 各命令のサイクルコストを Mednafen 準拠に更新 (MOVS=5, CMPS=9, STOS=3, LODS=3, SCAS=4, INS=6, OUTS=7)

### 5d. EEPROM/セーブ（実装済み）
- **内部 EEPROM (iEEPROM)**: 1KB、ポート 0xBA-0xBE。オーナー名・生年月日等のシステム設定
- **ゲーム EEPROM (wsEEPROM)**: 128B/1KB/2KB、ポート 0xC4-0xC8。ROM ヘッダ SaveSizeCode で判定
  - 0x10 → 128 bytes, 0x20 → 2048 bytes, 0x50 → 1024 bytes
  - それ以外は従来の SRAM（メモリマップ 0x10000-0x1FFFF）
- **ワードアドレッシング**: アドレスレジスタは 16-bit ワード単位、`address << 1` でバイトオフセット
- **即時アクセス**: Mednafen 準拠でコマンド遅延なし（ステータスは常に ready）
- **セーブ永続化**: `.sav` ファイル（ROM パスから拡張子変更）。起動時ロード・終了時セーブ
- **内部 EEPROM 初期化**: デフォルト名 "WONDERSWAN"、誕生日 2000/01/01 (BCD)

### 6a. スキャンライン実行分割（実装済み）
- Mednafen 準拠で 256 サイクル/スキャンラインを 128+96+32 の 3 セグメントに分割
- セグメント間イベント: Sound DMA #2 (128後), LineCompare IRQ4 (224後)
- `runCPUCycles()` ヘルパー関数に抽出

### 6b. LCDVtotal (port 0x16)（実装済み）
- フレームあたりの総ライン数を動的に変更可能
- `TotalLinesForFrame()`: `max(143, LCDVtotal) + 1` (Mednafen 準拠)
- RTC も動的ライン数に基づくサイクル数で進行

### 6c. シリアル通信スタブ（実装済み）
- `pkg/ws/serial.go` 新規。Mednafen comm.cpp 準拠のシングルプレイヤーモード
- Port 0xB1 (data), 0xB3 (control/status)
- `Process()`: 毎スキャンライン呼び出し、TX は即座に完了して IRQ0 発火
- RX は外部デバイスなしのためデータ到着せず
- シリアル待ちでハングするゲームが正常にフォールバック可能に

### 6d. APU Mednafen 準拠修正（実装済み）
- **Noise LFSR**: 右シフト→左シフト、feedback に XOR 1 追加（Mednafen 準拠）
- **Noise LFSR リセット**: 0x7FFF → 0 に修正
- **NoiseCfg マスク**: `val & 0x17` (bit3 はワンショットリセット、保存しない)
- **Noise タップテーブル**: `{14,13,12,11,10,9,8,7}` → `{14,10,13,4,8,6,9,11}` (Mednafen NoiseByetable)
- **Sweep 周期**: `CPUClock/128*SweepTime` → `8192*(SweepStep+1)` サイクル (Mednafen 準拠)
- **Sweep port 0x8D 書込**: divider=8192, counter=step+1 にリセット
- **Volume 計算**: `(sample-8)*128*vol/15` → `raw_nibble(0-15)*vol(0-15)` (Mednafen 準拠)
- **マスタースケール**: ミキサーで ×20 してint16レンジに合わせ (Mednafen Blip_Synth volume(2.5) 相当)
- **Voice D/A**: `(val-128)*64` → Mednafen 準拠の raw 値 (0-255) でミキサーに渡す
- **SCR1 カラーモード色0**: palette bit2 ルール適用 → 無条件不透明に修正
- **HyperVoice ミキシングスケール**: masterScale 未適用 → 通常チャンネルと同じスケール適用 (Mednafen 準拠)

## リファレンスソース
- **Mednafen**: `https://github.com/libretro-mirrors/mednafen-git/tree/master/src/wswan`
  - `gfx.cpp`: PPU レンダリング（最重要リファレンス）
  - `memory.cpp`: I/O ポートハンドリング
  - `interrupt.cpp`: 割り込み処理
  - `tcache.cpp`: タイルキャッシュ/デコード
- **Oswan**: `/Users/suzuki.kentaro/IdeaProjects/ws-go/Oswan173/` (バイナリのみ、ソースなし)

## ModRM Address Caching (CRITICAL)
`resolveModRM(mod, rm)` must be called once after `decodeModRM()` before any `readModRM`/`writeModRM`. This caches the effective address in `cpu.modrmSeg`/`cpu.modrmOff`. Without this, `getModRMAddress()` consumes displacement bytes from the instruction stream, and a read+write pair would fetch them twice, corrupting subsequent instructions.

## Audio Pipeline

### アーキテクチャ
```
HBlank ISR (CPU) → port 0x89 write → APU.ChannelVolume[1]
                                        ↓
APU.Tick() → Voice D/A mode (SoundCtrl bit5) → signed PCM (-128..+127)
           → Normal wavetable channels → signed nibble (-8..+7) × volume
           → BlipBuf.AddDelta(cyclePos, delta) at exact CPU cycle
                                        ↓
APU.EndScanline(256) → BlipBuf.EndFrame(256) + ReadSamples (~4 samples)
                     → integrator drift correction
                     → Ring buffer (32768 samples, overwrite on full)
                                        ↓
Stream.Read() (Ebitengine audio thread) → stereo duplicate → speakers
```

### Key Parameters
- **SampleRate**: 48,000 Hz (CPUClock の正除数: 3,072,000 / 48,000 = 64 cycles/sample)
- **TPS**: `ebiten.SyncWithFPS` — 表示リフレッシュレートに同期。フレームアキュムレータで75.47fpsエミュレーション維持
- **Ring buffer**: 32768 samples (~682ms)。満杯時は最古サンプル上書き
- **BlipBuf**: Blackman窓sinc帯域制限合成（16tap×32phase）。スキャンラインごとに EndFrame+ReadSamples
- **DC センタリング**: Voice D/A: PCM-128, Wavetable: raw-8, Noise: {+7,-8}。BlipBuf インテグレータドリフト防止
- **インテグレータ補正**: 各スキャンライン後に `integ = lastBlipMono` で固定小数点丸め誤差を除去

### Bad Apple の音声フロー
1. ROM init: SoundCtrl=0x22 (bit5=Voice D/A, bit1=CH2 enable), VoiceVolume=0x0F (L/R full)
2. HBlank ISR (毎スキャンライン): ROM1Bank から 1 byte 読み → OUT 0x89
3. APU Voice D/A: (ChannelVolume[1] - 128) を signed PCM として使用
4. signed PCM × masterScale(20) → BlipBuf delta → band-limited resample → ring buffer

### 表示同期
- `ebiten.SetTPS(ebiten.SyncWithFPS)` で表示リフレッシュレートに同期
- `frontend/game.go` のフレームアキュムレータが時間を蓄積し、1-2 RunFrame/Update を実行
- 旧 TPS=76 方式ではフレームスキップによるちらつきが発生していた

### Stream.Read のサンプル保持
- リングバッファ空時に無音(0)ではなく直前のサンプルを保持 (sample-and-hold)
- バッファアンダーラン時のクリックノイズを防止

### Ebitengine audio.Player の GC 防止
- `frontend/audio.go` の `audioPlayer` をパッケージレベル変数に保持
- ローカル変数のみだと ~1秒後に GC が回収し再生停止するバグがあった

## Debugging Tips
- `cmd/framecheck/main.go` でヘッドレスフレーム生成（Ebitengine 不要）
- MapBase 変更頻度でデコンプレッション速度を測定可能（2-3 frames/swap が正常）
- IRAM[0x2000-0x3FFF] の非ゼロバイト数でタイルデータ量を確認
- `WebFetch` で Mednafen raw ソースを直接取得可能
