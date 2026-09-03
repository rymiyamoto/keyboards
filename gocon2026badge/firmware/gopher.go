package main

import (
	"tinygo.org/x/drivers/pixel"
	"tinygo.org/x/drivers/st7789"
)

// gopher は右上の画面外から中央へ飛来し、しばらく上下に漂った後、
// 左へ抜けて画面外でひと休みしてから繰り返す。
// マーキー帯 (y >= marqueeY) には入らないこと。
const (
	gopherW = 100
	gopherH = 100

	gopherHomeX = 60 // 漂うときの位置
	gopherHomeY = 60

	gopherEnterSteps = 60 // 進入にかける更新回数 (66.7ms/回 ≈ 4 秒)
	gopherBobSteps   = 64 // 漂う時間 (yOffsets 2 周 ≈ 4.3 秒)
	gopherExitSpeed  = 4  // 退場速度 px/回 (揺れが見えるよう控えめに ≈ 2.7 秒)
	gopherWaitSteps  = 15 // 画面外での待機 ≈ 1 秒
)

type gopherPhase int

const (
	gopherEnter gopherPhase = iota
	gopherBob
	gopherExit
	gopherWait
)

var (
	gopherState       = gopherEnter
	gopherStep        = 0
	gopherOsc         = 0 // 揺れの位相 (bob と exit で共通に進め、つなぎ目で周期を変えない)
	gopherSlide       = 0 // exit の累積スライド量
	gopherX           = 240
	gopherY           = -gopherH
	gopherPrevY       = -gopherH
	gopherPrevVisible = false
)

// 上下の漂い (32 分割の疑似サイン波)
var yOffsets = [...]int{
	0, 2, 3, 4, 6, 7, 7, 8,
	8, 8, 7, 7, 6, 4, 3, 2,
	0, -2, -3, -4, -6, -7, -7, -8,
	-8, -8, -7, -7, -6, -4, -3, -2,
}

// gopherSwayX は左右の揺れ (最大 ±4px)。位相は上下の揺れの 1/4 遅れで、
// bob 開始から半周期かけて振幅を 0 → 最大へ立ち上げる
func gopherSwayX() int {
	amp := gopherOsc
	if amp > 16 {
		amp = 16
	}
	return yOffsets[(gopherOsc+8)%len(yOffsets)] * amp / 32
}

// updateGopher はアニメーションを 1 ステップ進めて描画する
func updateGopher(display st7789.Device) error {
	switch gopherState {
	case gopherEnter:
		// 右上 (240, -100) から中央へ、木の葉のように揺れながら降りてくる。
		// 揺れは中央に近づくほど減衰し、着地点で漂い (bob) に連続する
		remain := gopherEnterSteps - gopherStep
		swayX := yOffsets[(gopherStep*2)%len(yOffsets)] * 2 * remain / gopherEnterSteps
		swayY := yOffsets[(gopherStep*2+8)%len(yOffsets)] * remain / gopherEnterSteps
		gopherX = 240 + (gopherHomeX-240)*gopherStep/gopherEnterSteps + swayX
		gopherY = -gopherH + (gopherHomeY+gopherH)*gopherStep/gopherEnterSteps + swayY
		if gopherStep >= gopherEnterSteps {
			gopherState = gopherBob
			gopherStep = 0
			gopherOsc = 0
		}
	case gopherBob:
		// 上下の漂いに、位相を 1/4 ずらした左右の揺れ (±4px) を重ねて楕円軌道にする。
		// 左右の振幅は 0 から立ち上げて進入からの座標の飛びをなくす
		gopherX = gopherHomeX + gopherSwayX()
		gopherY = gopherHomeY + yOffsets[gopherOsc%len(yOffsets)]
		gopherOsc++
		if gopherStep >= gopherBobSteps {
			gopherState = gopherExit
			gopherStep = 0
			gopherSlide = 0
		}
	case gopherExit:
		// 漂いと同じリズムで揺れ続けながら、0 から加速するスライドで左へ抜けていく
		vx := gopherStep / 2
		if vx > gopherExitSpeed {
			vx = gopherExitSpeed
		}
		gopherSlide += vx
		gopherX = gopherHomeX + gopherSwayX() - gopherSlide
		gopherY = gopherHomeY + yOffsets[gopherOsc%len(yOffsets)]
		gopherOsc++
		if gopherX <= -gopherW {
			gopherState = gopherWait
			gopherStep = 0
		}
	case gopherWait:
		if gopherStep >= gopherWaitSteps {
			gopherState = gopherEnter
			gopherStep = 0
			gopherX = 240
			gopherY = -gopherH
		}
	}
	gopherStep++

	return drawGopher(display)
}

// overlayGopher は pixelBuf へ gopher を透過合成する (0x0000 を透明色とみなす)。
// 画面外とマーキー帯へのはみ出しはクリップする。
func overlayGopher(xofs, yofs int) {
	b := gopher565
	for y := 0; y < gopherH; y++ {
		sy := y + yofs
		if sy < 0 || sy >= marqueeY {
			continue
		}
		for x := 0; x < gopherW; x++ {
			sx := x + xofs
			if sx < 0 || sx >= 240 {
				continue
			}
			p := (uint16(b[(x+y*gopherW)*2+1]) << 8) + uint16(b[(x+y*gopherW)*2+0])
			if p != 0x0000 {
				pixelBuf.Set(sx, sy, pixel.RGB565BE(p))
			}
		}
	}
}

// drawGopher は前回位置と今回位置を覆う帯だけを背景から再合成して転送する
func drawGopher(display st7789.Device) error {
	visible := gopherX > -gopherW && gopherX < 240 &&
		gopherY > -gopherH && gopherY < marqueeY
	if !visible && !gopherPrevVisible {
		// 画面外に居続けている間は転送しない
		gopherPrevY = gopherY
		return nil
	}

	// 前の DMA 転送が pixelBuf を読んでいる間は書き換えない
	spiBus.Wait()

	top := gopherY
	if gopherPrevY < top {
		top = gopherPrevY
	}
	bottom := gopherY + gopherH
	if gopherPrevY+gopherH > bottom {
		bottom = gopherPrevY + gopherH
	}
	if top < 0 {
		top = 0
	}
	if bottom > marqueeY {
		bottom = marqueeY
	}
	gopherPrevY = gopherY
	gopherPrevVisible = visible
	if bottom <= top {
		return nil
	}

	raw := pixelBuf.RawBuffer()
	copy(raw[top*240*2:bottom*240*2], background565[top*240*2:bottom*240*2])
	overlayGopher(gopherX, gopherY)

	band := pixel.NewImageFromBytes[pixel.RGB565BE](240, bottom-top, raw[top*240*2:bottom*240*2])
	return display.DrawBitmap(0, int16(top), band)
}
