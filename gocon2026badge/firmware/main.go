package main

import (
	"embed"
	_ "embed"
	"image/color"
	"machine"
	"time"

	pio "github.com/tinygo-org/pio/rp2-pio"
	"github.com/tinygo-org/pio/rp2-pio/piolib"
	"tinygo.org/x/drivers/pixel"
	"tinygo.org/x/drivers/st7789"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/freesans"
)

//go:embed images
var imageDir embed.FS

func main() {
	err := run()
	for err != nil {
		println(err)
		time.Sleep(1 * time.Second)
	}
}

func writeColors(s pio.StateMachine, ws *piolib.WS2812B, colors []uint32) {
	ws.WriteRaw(colors)
}

const (
	white = 0x0F0F0FFF
	red   = 0x000F00FF
	green = 0x0F0000FF
	blue  = 0x00000FFF
	black = 0x000000FF
)

func run() error {
	machine.SPI1.Configure(machine.SPIConfig{
		Frequency: 20000000,
		Mode:      0,
		SCK:       machine.GPIO10,
		SDO:       machine.GPIO11,
		SDI:       machine.GPIO12, // 一旦ダミーで設定する
	})

	display := st7789.New(machine.SPI1,
		machine.GPIO15, // RESET
		machine.GPIO14, // DC
		machine.GPIO13, // CS
		machine.GPIO12) // BL

	display.Configure(st7789.Config{
		Height: 240,
		Width:  240,
	})

	// Clear the screen to white
	display.FillScreen(color.RGBA{0x00, 0x00, 0x00, 0xFF})
	display.SetRotation(st7789.ROTATION_90)

	tinyfont.WriteLine(&display, &freesans.Bold12pt7b, 00, 50, "Hello", color.RGBA{R: 255, G: 255, B: 0, A: 255})
	tinyfont.WriteLine(&display, &freesans.Bold12pt7b, 00, 80, "Gophers!", color.RGBA{R: 255, G: 0, B: 255, A: 255})

	err := initImage()
	if err != nil {
		return err
	}
	err = drawImage(display)
	if err != nil {
		return err
	}

	btnA := machine.GPIO3
	btnB := machine.GPIO6
	btnR := machine.GPIO7
	btnU := machine.GPIO8
	btnL := machine.GPIO28
	btnD := machine.GPIO29

	buttons := []machine.Pin{btnA, btnB, btnR, btnU, btnL, btnD}
	for _, b := range buttons {
		b.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	}

	wsPin := machine.GPIO9
	//wsPin := machine.GPIO16
	s, _ := pio.PIO0.ClaimStateMachine()
	ws, _ := piolib.NewWS2812B(s, wsPin)
	err = ws.EnableDMA(true)
	if err != nil {
		return err
	}
	wsLeds := [16]uint32{}
	for i := range wsLeds {
		wsLeds[i] = black
	}
	writeColors(s, ws, wsLeds[:])

	ledBuffer[0x00] = 0x02021FFF
	ledBuffer[0x01] = 0x02020FFF
	ledBuffer[0x02] = 0x020208FF
	ledBuffer[0x03] = 0x020204FF
	ledBuffer[0x04] = 0x020202FF
	ledBuffer[0x05] = 0x020202FF
	ledBuffer[0x06] = 0x020202FF
	ledBuffer[0x07] = 0x020202FF
	ledBuffer[0x08] = 0x1F0202FF
	ledBuffer[0x09] = 0x0F0202FF
	ledBuffer[0x0A] = 0x080202FF
	ledBuffer[0x0B] = 0x040202FF
	ledBuffer[0x0C] = 0x020202FF
	ledBuffer[0x0D] = 0x020202FF
	ledBuffer[0x0E] = 0x020202FF
	ledBuffer[0x0F] = 0x020202FF

	ticker := time.Tick(8 * time.Millisecond)
	cnt := 0
	for {
		<-ticker

		switch cnt % 10 {
		case 0, 5:
			//UpdateRainbowChase(cnt)
			//UpdateMeteor(cnt)
			rotate(true)
		case 1, 6:
			writeColors(s, ws, ledBuffer[:])
		}

		cnt++
	}

	return nil
}

func rotate(right bool) [NumLEDs]uint32 {
	if right {
		tmp := ledBuffer[0]
		for i := range ledBuffer[:] {
			ledBuffer[i] = ledBuffer[(i+1)%NumLEDs]
		}
		ledBuffer[NumLEDs-1] = tmp
	} else {
		tmp := ledBuffer[NumLEDs-1]
		for i := range ledBuffer[:] {
			ledBuffer[(i+1)%NumLEDs] = ledBuffer[i]
		}
		ledBuffer[0] = tmp
	}

	return ledBuffer
}

const (
	NumLEDs = 16
)

var ledBuffer [NumLEDs]uint32

// 0xGGRRBBAA の形にパックするヘルパー関数
func toGGRRBBAA(g, r, b, a uint8) uint32 {
	return (uint32(g) << 24) | (uint32(r) << 16) | (uint32(b) << 8) | uint32(a)
}

// 簡単なHSVからRGBへの変換関数（レインボー用）
// hue: 0-359, sat: 0-255, val: 0-255
func hsvToRGB(h int, s, v uint8) (r, g, b uint8) {
	if s == 0 {
		return v, v, v
	}

	f := float32(h%60) / 60.0
	p := uint8(float32(v) * (1.0 - float32(s)/255.0))
	q := uint8(float32(v) * (1.0 - f*float32(s)/255.0))
	t := uint8(float32(v) * (1.0 - (1.0-f)*float32(s)/255.0))

	switch h / 60 {
	case 0:
		return v, t, p
	case 1:
		return q, v, p
	case 2:
		return p, v, t
	case 3:
		return p, q, v
	case 4:
		return t, p, v
	default:
		return v, p, q
	}
}

func UpdateRainbowChase(step int) [NumLEDs]uint32 {
	for i := 0; i < NumLEDs; i++ {
		// 位置と時間(step)を組み合わせて色相を決定
		hue := (i*360/NumLEDs + step*5) % 360
		r, g, b := hsvToRGB(hue, 255, 255)

		// アルファ（輝度）はマックスの0xFF
		ledBuffer[i] = toGGRRBBAA(g, r, b, 0xFF)
	}
	return ledBuffer
}

func UpdateMeteor(step int) [NumLEDs]uint32 {
	// 彗星の現在の先頭位置
	meteorPos := step % NumLEDs

	// 基本の彗星の色（例：クールなアクアブルー）
	baseR, baseG, baseB := uint8(0), uint8(200), uint8(255)

	for i := 0; i < NumLEDs; i++ {
		// 先頭位置からの距離（逆方向の残像を計算）
		diff := (meteorPos - i + NumLEDs) % NumLEDs

		var fade float64
		if diff == 0 {
			fade = 1.0 // 先頭は一番明るい
		} else if diff < 6 {
			// 後ろの5個まで残像を残す
			fade = 1.0 - (float64(diff) * 0.18)
		} else {
			fade = 0.0 // それ以降は消灯
		}

		r := uint8(float64(baseR) * fade)
		g := uint8(float64(baseG) * fade)
		b := uint8(float64(baseB) * fade)

		// 残像の減衰に合わせてアルファも絞ると綺麗です
		a := uint8(255 * fade)

		ledBuffer[i] = toGGRRBBAA(g, r, b, a)
	}
	return ledBuffer
}

var (
	xxx = pixel.NewImage[pixel.RGB565BE](240, 240)
)

func initImage() error {
	{
		b, err := imageDir.ReadFile("images/gocon2026.rgb565")
		//b, err := imageDir.ReadFile("images/background.rgb565")
		//b, err := imageDir.ReadFile("images/background.rgb232")
		if err != nil {
			return err
		}

		for y := 0; y < 240; y++ {
			for x := 0; x < 240; x++ {
				p := uint16(b[(x+y*240)*2+1])<<8 + uint16(b[(x+y*240)*2+0])

				// RGB232
				// // rrgggbb
				// r := (b[x+y*240] >> 5) & 0x03
				// g := (b[x+y*240] >> 2) & 0x07
				// b := (b[x+y*240] >> 0) & 0x03
				// // rr000ggg000bb000
				// //   14    9    3
				// p := uint16(r)<<13 + uint16(g)<<9 + uint16(b)<<3
				xxx.Set(x, y, pixel.RGB565BE(p))
			}
		}
	}

	if false {
		b, err := imageDir.ReadFile("images/gopher.rgb565")
		if err != nil {
			return err
		}

		for y := 0; y < 100; y++ {
			for x := 0; x < 100; x++ {
				p := (uint16(b[(x+y*100)*2+1]) << 8) + uint16(b[(x+y*100)*2+0])
				xxx.Set(x, y, pixel.RGB565BE(p))
			}
		}
	}

	//{
	//	b, err := imageDir.ReadFile("images/gopher.rgb565")
	//	if err != nil {
	//		return err
	//	}

	//	for y := 0; y < 100; y++ {
	//		for x := 0; x < 100; x++ {
	//			p := (uint16(b[(x+y*100)*2+1]) << 8) + uint16(b[(x+y*100)*2+0])
	//			xxx.Set(x, y, pixel.RGB565BE(p))
	//		}
	//	}
	//}

	return nil
}

func drawImage(display st7789.Device) error {
	//display.DrawRGBBitmap8(0, 0, badgeImage, 240, 240)

	display.DrawBitmap(0, 0, xxx)
	//display.DrawBitmap(0, 0, xxx)
	//time.Sleep(1 * time.Second)
	//display.DrawBitmap(0, 0, yyy)

	return nil
}
