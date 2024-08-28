package main

import (
	"image"
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// -- sprite

type SpriteAnchor int

const (
	// AnchorBottom anchors the bottom of the sprite to its Z-position
	AnchorBottom SpriteAnchor = iota
	// AnchorCenter anchors the center of the sprite to its Z-position
	AnchorCenter
	// AnchorTop anchors the top of the sprite to its Z-position
	AnchorTop
)

func getAnchorVerticalOffset(anchor SpriteAnchor, spriteScale float64, cameraHeight int) float64 {
	halfHeight := float64(cameraHeight) / 2

	switch anchor {
	case AnchorBottom:
		return halfHeight - (spriteScale * halfHeight)
	case AnchorCenter:
		return halfHeight
	case AnchorTop:
		return halfHeight + (spriteScale * halfHeight)
	}

	return 0
}

type Sprite struct {
	*Entity
	w, h           int
	animationRate  int
	isFocusable    bool
	illumination   float64
	animReversed   bool
	animCounter    int
	loopCounter    int
	columns, rows  int
	texNum, lenTex int
	texFacingMap   map[float64]int
	texFacingKeys  []float64
	texRects       []image.Rectangle
	textures       []*ebiten.Image
	screenRect     *image.Rectangle
}

func (s *Sprite) getScale() float64 {
	return s.Entity.scale
}

func (s *Sprite) getVerticalAnchor() SpriteAnchor {
	return s.Entity.verticalAnchor
}

func (s *Sprite) Texture() *ebiten.Image {
	return s.textures[s.texNum]
}

func (s *Sprite) TextureRect() image.Rectangle {
	return s.texRects[s.texNum]
}

func (s *Sprite) Illumination() float64 {
	return s.illumination
}

func (s *Sprite) SetScreenRect(rect *image.Rectangle) {
	s.screenRect = rect
}

func (s *Sprite) IsFocusable() bool {
	return s.isFocusable
}

func NewSprite(
	x, y, scale float64, img *ebiten.Image, mapColor color.RGBA,
	anchor SpriteAnchor, collisionRadius, collisionHeight float64,
) *Sprite {
	s := &Sprite{
		Entity: &Entity{
			pos:             &Vec2{X: x, Y: y},
			posZ:            0,
			scale:           scale,
			verticalAnchor:  anchor,
			angle:           0,
			velocity:        0,
			collisionRadius: collisionRadius,
			collisionHeight: collisionHeight,
			mapColor:        mapColor,
		},
		isFocusable: true,
	}

	s.texNum = 0
	s.lenTex = 1
	s.textures = make([]*ebiten.Image, s.lenTex)

	s.w, s.h = img.Bounds().Dx(), img.Bounds().Dy()
	s.texRects = []image.Rectangle{image.Rect(0, 0, s.w, s.h)}

	s.textures[0] = img

	return s
}

func NewSpriteFromSheet(
	x, y, scale float64, img *ebiten.Image, mapColor color.RGBA,
	columns, rows, spriteIndex int, anchor SpriteAnchor, collisionRadius, collisionHeight float64,
) *Sprite {
	s := &Sprite{
		Entity: &Entity{
			pos:             &Vec2{X: x, Y: y},
			posZ:            0,
			scale:           scale,
			verticalAnchor:  anchor,
			angle:           0,
			velocity:        0,
			collisionRadius: collisionRadius,
			collisionHeight: collisionHeight,
			mapColor:        mapColor,
		},
		isFocusable: true,
	}

	s.texNum = spriteIndex
	s.columns, s.rows = columns, rows
	s.lenTex = columns * rows
	s.textures = make([]*ebiten.Image, s.lenTex)
	s.texRects = make([]image.Rectangle, s.lenTex)

	w, h := img.Bounds().Dx(), img.Bounds().Dy()

	// crop sheet by given number of columns and rows into a single dimension array
	s.w = w / columns
	s.h = h / rows

	for r := 0; r < rows; r++ {
		y := r * s.h
		for c := 0; c < columns; c++ {
			x := c * s.w
			cellRect := image.Rect(x, y, x+s.w, y+s.h)
			cellImg := img.SubImage(cellRect).(*ebiten.Image)

			index := c + r*columns
			s.textures[index] = cellImg
			s.texRects[index] = cellRect
		}
	}

	return s
}

func NewAnimatedSprite(
	x, y, scale float64, animationRate int, img *ebiten.Image, mapColor color.RGBA,
	columns, rows int, anchor SpriteAnchor, collisionRadius, collisionHeight float64,
) *Sprite {
	s := &Sprite{
		Entity: &Entity{
			pos:             &Vec2{X: x, Y: y},
			posZ:            0,
			scale:           scale,
			verticalAnchor:  anchor,
			angle:           0,
			velocity:        0,
			collisionRadius: collisionRadius,
			collisionHeight: collisionHeight,
			mapColor:        mapColor,
		},
		isFocusable: true,
	}

	s.animationRate = animationRate
	s.animCounter = 0
	s.loopCounter = 0

	s.texNum = 0
	s.columns, s.rows = columns, rows
	s.lenTex = columns * rows
	s.textures = make([]*ebiten.Image, s.lenTex)
	s.texRects = make([]image.Rectangle, s.lenTex)

	w, h := img.Bounds().Dx(), img.Bounds().Dy()

	// crop sheet by given number of columns and rows into a single dimension array
	s.w = w / columns
	s.h = h / rows

	for r := 0; r < rows; r++ {
		y := r * s.h
		for c := 0; c < columns; c++ {
			x := c * s.w
			cellRect := image.Rect(x, y, x+s.w, y+s.h)
			cellImg := img.SubImage(cellRect).(*ebiten.Image)

			index := c + r*columns
			s.textures[index] = cellImg
			s.texRects[index] = cellRect
		}
	}

	return s
}

func (s *Sprite) SetTextureFacingMap(texFacingMap map[float64]int) {
	s.texFacingMap = texFacingMap

	// create pre-sorted list of keys used during facing determination
	s.texFacingKeys = make([]float64, len(texFacingMap))
	for k := range texFacingMap {
		s.texFacingKeys = append(s.texFacingKeys, k)
	}
	sort.Float64s(s.texFacingKeys)
}

func (s *Sprite) getTextureFacingKeyForAngle(facingAngle float64) float64 {
	var closestKeyAngle float64 = -1
	if s.texFacingMap == nil || len(s.texFacingMap) == 0 || s.texFacingKeys == nil || len(s.texFacingKeys) == 0 {
		return closestKeyAngle
	}

	closestKeyDiff := math.MaxFloat64
	for _, keyAngle := range s.texFacingKeys {
		keyDiff := math.Min(Pi2-math.Abs(float64(keyAngle)-facingAngle), math.Abs(float64(keyAngle)-facingAngle))
		if keyDiff < closestKeyDiff {
			closestKeyDiff = keyDiff
			closestKeyAngle = keyAngle
		}
	}

	return closestKeyAngle
}

func (s *Sprite) SetAnimationReversed(isReverse bool) {
	s.animReversed = isReverse
}

func (s *Sprite) SetAnimationFrame(texNum int) {
	s.texNum = texNum
}

func (s *Sprite) ResetAnimation() {
	s.animCounter = 0
	s.loopCounter = 0
	s.texNum = 0
}

func (s *Sprite) LoopCounter() int {
	return s.loopCounter
}

func (s *Sprite) ScreenRect() *image.Rectangle {
	return s.screenRect
}

func (s *Sprite) Update(camPos *Vec2) {
	if s.animationRate <= 0 {
		return
	}

	if s.animCounter >= s.animationRate {
		minTexNum := 0
		maxTexNum := s.lenTex - 1

		if len(s.texFacingMap) > 1 && camPos != nil {
			// TODO: may want to be able to change facing even between animation frame changes

			// use facing from camera position to determine min/max texNum in texFacingMap
			// to update facing of sprite relative to camera and sprite angle
			texRow := 0

			// calculate angle from sprite relative to camera position by getting angle of line between them
			lineToCam := Line{X1: s.pos.X, Y1: s.pos.Y, X2: camPos.X, Y2: camPos.Y}
			facingAngle := lineToCam.angle() - s.angle
			if facingAngle < 0 {
				// convert to positive angle needed to determine facing index to use
				facingAngle += Pi2
			}
			facingKeyAngle := s.getTextureFacingKeyForAngle(facingAngle)
			if texFacingValue, ok := s.texFacingMap[facingKeyAngle]; ok {
				texRow = texFacingValue
			}

			minTexNum = texRow * s.columns
			maxTexNum = texRow*s.columns + s.columns - 1
		}

		s.animCounter = 0

		if s.animReversed {
			s.texNum -= 1
			if s.texNum > maxTexNum || s.texNum < minTexNum {
				s.texNum = maxTexNum
				s.loopCounter++
			}
		} else {
			s.texNum += 1
			if s.texNum > maxTexNum || s.texNum < minTexNum {
				s.texNum = minTexNum
				s.loopCounter++
			}
		}
	} else {
		s.animCounter++
	}
}

func (s *Sprite) AddDebugLines(lineWidth int, clr color.Color) {
	lW := float64(lineWidth)
	sW := float64(s.w)
	sH := float64(s.h)
	sCr := s.collisionRadius * sW

	for i, img := range s.textures {
		imgRect := s.texRects[i]
		x, y := float64(imgRect.Min.X), float64(imgRect.Min.Y)

		// bounding box
		ebitenutil.DrawRect(img, x, y, lW, sH, clr)
		ebitenutil.DrawRect(img, x, y, sW, lW, clr)
		ebitenutil.DrawRect(img, x+sW-lW-1, y+sH-lW-1, lW, -sH, clr)
		ebitenutil.DrawRect(img, x+sW-lW-1, y+sH-lW-1, -sW, lW, clr)

		// center lines
		ebitenutil.DrawRect(img, x+sW/2-lW/2-1, y, lW, sH, clr)
		ebitenutil.DrawRect(img, x, y+sH/2-lW/2-1, sW, lW, clr)

		// collision markers
		if s.collisionRadius > 0 {
			ebitenutil.DrawRect(img, x+sW/2-sCr-lW/2-1, y, lW, sH, color.White)
			ebitenutil.DrawRect(img, x+sW/2+sCr-lW/2-1, y, lW, sH, color.White)
		}
	}
}

func (sprite *Sprite) drawSpriteBox(screen *ebiten.Image) {
	r := sprite.ScreenRect()
	if r == nil {
		return
	}

	minX, minY := float32(r.Min.X), float32(r.Min.Y)
	maxX, maxY := float32(r.Max.X), float32(r.Max.Y)

	vector.StrokeRect(screen, minX, minY, maxX-minX, maxY-minY, 1, color.RGBA{255, 0, 0, 255}, false)
}

func (s *Sprite) drawSpriteIndicator(screen *ebiten.Image) {
	r := s.ScreenRect()
	if r == nil {
		return
	}

	dX, dY := float32(r.Dx())/8, float32(r.Dy())/8
	midX, minY := float32(r.Max.X)-float32(r.Dx())/2, float32(r.Min.Y)-dY

	vector.StrokeLine(screen, midX, minY+dY, midX-dX, minY, 1, color.RGBA{0, 255, 0, 255}, false)
	vector.StrokeLine(screen, midX, minY+dY, midX+dX, minY, 1, color.RGBA{0, 255, 0, 255}, false)
	vector.StrokeLine(screen, midX-dX, minY, midX+dX, minY, 1, color.RGBA{0, 255, 0, 255}, false)
}
