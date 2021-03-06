package tracker

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"log"
	"strconv"

	"github.com/golang/freetype/truetype"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
)

type Tracker struct {
	pos      image.Point
	size     image.Point
	hintPos  image.Point
	hintSize image.Point

	background     *ebiten.Image
	backgroundHelp *ebiten.Image
	font           font.Face
	fontSmall      font.Face
	sheetDisabled  *ebiten.Image
	sheetEnabled   *ebiten.Image

	items       []Item
	zoneItemMap ZoneItemMap
	locations   []string
	input       kbInput

	woths     []string
	barrens   []string
	always    [7]string // skull, bigg, 30, 40, 50, OOT, frogs 2
	sometimes []string  // freeform input

	undoStack []undoStackEntry
	redoStack []undoStackEntry
}

const (
	capacityFontSize = 20
	templeFontSize   = 13
)

type ZoneItemMap [9][9]string

func New(
	dimensions image.Rectangle,
	hintDimensions image.Rectangle,
	items []Item,
	zoneItemMap ZoneItemMap,
	locations []string,
) (*Tracker, error) {
	background, _, err := ebitenutil.NewImageFromFile("assets/background.png", ebiten.FilterDefault)
	if err != nil {
		return nil, err
	}

	backgroundHelp, _, err := ebitenutil.NewImageFromFile("assets/background-help.png", ebiten.FilterDefault)
	if err != nil {
		return nil, err
	}

	ttf, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return nil, err
	}

	sheetDisabled, _, err := ebitenutil.NewImageFromFile("assets/items-disabled.png", ebiten.FilterDefault)
	if err != nil {
		return nil, err
	}

	sheetEnabled, _, err := ebitenutil.NewImageFromFile("assets/items.png", ebiten.FilterDefault)
	if err != nil {
		return nil, err
	}

	tracker := &Tracker{
		pos:            dimensions.Min,
		size:           dimensions.Size(),
		hintPos:        hintDimensions.Min,
		hintSize:       hintDimensions.Size(),
		items:          items,
		locations:      locations,
		zoneItemMap:    zoneItemMap,
		background:     background,
		backgroundHelp: backgroundHelp,
		sheetDisabled:  sheetDisabled,
		sheetEnabled:   sheetEnabled,
		font: truetype.NewFace(ttf, &truetype.Options{
			Size:    capacityFontSize,
			Hinting: font.HintingFull,
		}),
		fontSmall: truetype.NewFace(ttf, &truetype.Options{
			Size:    templeFontSize,
			Hinting: font.HintingFull,
		}),
	}

	tracker.changeItem(tracker.getItemIndexByName("Gold Skulltula Token"), true)
	tracker.changeItem(tracker.getItemIndexByName("Kokiri Tunic"), true)
	tracker.changeItem(tracker.getItemIndexByName("Kokiri Boots"), true)

	return tracker, nil
}

func (tracker *Tracker) GetZoneItem(zoneKP, itemKP int) (string, error) {
	if zoneKP <= 0 || zoneKP > 9 {
		return "", errors.New("invalid zoneKP, must be [1-9]")
	}
	if itemKP <= 0 || itemKP > 9 {
		return "", errors.New("invalid itemKP, must be [1-9]")
	}

	name := tracker.zoneItemMap[zoneKP-1][itemKP-1]
	if name == "" {
		return "", fmt.Errorf("no item defined for zone %d item %d", zoneKP, itemKP)
	}

	return name, nil
}

func (tracker *Tracker) GetZoneItemIndex(zoneKP, itemKP int) (int, error) {
	name, err := tracker.GetZoneItem(zoneKP, itemKP)
	if err != nil {
		return -1, err
	}

	index := tracker.getItemIndexByName(name)
	if index < 0 {
		return -1, fmt.Errorf("item name misconfigured: %s", name)
	}

	return index, nil
}

// getItemIndexByPos returns the index of the item at the given position or -1 if
// there is no item under the given pixel.
func (tracker *Tracker) getItemIndexByPos(x, y int) int {
	for k := range tracker.items {
		if (image.Point{x, y}).In(tracker.items[k].Rect()) {
			return k
		}
	}

	return -1
}

func (tracker *Tracker) getItemIndexByName(name string) int {
	for k := range tracker.items {
		if tracker.items[k].Name == name {
			return k
		}
	}

	return -1
}

// ClickLeft upgrades the item under the given point.
func (tracker *Tracker) ClickLeft(x, y int) {
	i := tracker.getItemIndexByPos(x, y)
	if i < 0 {
		return
	}

	tracker.changeItem(i, true)
}

// ClickRight downgrades the item under the given point.
func (tracker *Tracker) ClickRight(x, y int) {
	i := tracker.getItemIndexByPos(x, y)
	if i < 0 {
		return
	}

	tracker.changeItem(i, false)
}

func (tracker *Tracker) changeItem(itemIndex int, isUpgrade bool) {
	var fn func() bool
	if isUpgrade {
		fn = tracker.items[itemIndex].Upgrade
	} else {
		fn = tracker.items[itemIndex].Downgrade
	}

	if fn() {
		tracker.appendToUndoStack(itemIndex, isUpgrade)
	}
}

func (tracker *Tracker) Wheel(x, y int, up bool) {
	i := tracker.getItemIndexByPos(x, y)
	if i < 0 {
		return
	}

	switch {
	case tracker.items[i].IsMedallion:
		tracker.items[i].CycleTemple(up)
	default:
		if up {
			tracker.ClickLeft(x, y)
		} else {
			tracker.ClickRight(x, y)
		}
	}
}

func (tracker *Tracker) Draw(screen *ebiten.Image) {
	op := ebiten.DrawImageOptions{}
	drawState := func(state bool, sheet *ebiten.Image) {
		for k := range tracker.items {
			if tracker.items[k].Enabled != state {
				continue
			}

			pos := tracker.items[k].Rect().Min.Add(tracker.pos)
			op.GeoM.Reset()
			op.GeoM.Translate(float64(pos.X), float64(pos.Y))

			if err := screen.DrawImage(
				sheet.SubImage(tracker.items[k].SheetRect()).(*ebiten.Image),
				&op,
			); err != nil {
				log.Fatal(err)
			}
		}
	}

	_ = screen.DrawImage(tracker.background, nil)
	if tracker.kbInputStateIsAny(inputStateItemKPZoneInput, inputStateItemInput) {
		_ = screen.DrawImage(tracker.backgroundHelp, nil)
		if tracker.input.activeKPZone > 0 {
			tracker.drawActiveItemSlot(screen, tracker.input.activeKPZone)
		}
	}

	// Do two loops to avoid texture switches.
	drawState(false, tracker.sheetDisabled)
	drawState(true, tracker.sheetEnabled)

	tracker.drawTemples(screen)
	tracker.drawCapacities(screen)
	tracker.drawInputState(screen)
	tracker.drawHints(screen)
}

func (tracker *Tracker) drawActiveItemSlot(screen *ebiten.Image, slot int) {
	if slot <= 0 || slot > 9 {
		return
	}

	slot = []int{ // make maths ez
		0,
		6, 7, 8,
		3, 4, 5,
		0, 1, 2,
	}[slot]

	edge := gridSize * 3
	pos := image.Point{
		(slot % 3) * edge,
		(slot / 3) * edge,
	}
	size := image.Point{126, 126}

	if slot == 2 { // KP 9
		pos.Y = 0
		size.X = gridSize
		size.Y = 4*gridSize + (gridSize / 2)
	} else if slot == 8 { // KP 3
		pos.Y = 4*gridSize + (gridSize / 2)
		size.X = gridSize
		size.Y = pos.Y
	}

	ebitenutil.DrawRect(
		screen,
		float64(pos.X), float64(pos.Y),
		float64(size.X), float64(size.Y),
		color.RGBA{0xFF, 0xFF, 0xFF, 0x50},
	)
}

func (tracker *Tracker) drawTemples(screen *ebiten.Image) {
	for k := range tracker.items {
		if !tracker.items[k].IsMedallion {
			continue
		}

		rect := tracker.items[k].Rect()
		x, y := rect.Min.X, rect.Max.Y
		text.Draw(screen, tracker.items[k].TempleText(), tracker.fontSmall, x, y, color.White)
	}
}

func (tracker *Tracker) drawCapacities(screen *ebiten.Image) {
	for k := range tracker.items {
		var count int
		switch {
		case tracker.items[k].HasCapacity():
			count = tracker.items[k].Capacity()
		case tracker.items[k].IsCountable():
			count = tracker.items[k].Count()
		default:
			continue
		}

		if !tracker.items[k].Enabled {
			continue
		}

		rect := tracker.items[k].Rect()
		x, y := rect.Min.X, rect.Max.Y

		// HACK, display skull count centered on the right slot
		if tracker.items[k].Name == "Gold Skulltula Token" {
			x, y = rect.Min.X+gridSize+marginLeft, rect.Min.Y+marginTop+(gridSize/2)
			if count == 100 {
				x -= 5
			} else if count < 10 {
				x += 5
			}
		}

		str := strconv.Itoa(count)
		text.Draw(screen, str, tracker.font, x, y, color.White)
	}
}

func (tracker *Tracker) Reset(items []Item, zoneItemMap ZoneItemMap) {
	tracker.items = items
	tracker.zoneItemMap = zoneItemMap
	tracker.undoStack = tracker.undoStack[:0]
	tracker.redoStack = tracker.redoStack[:0]
	tracker.woths = tracker.woths[:0]
	tracker.barrens = tracker.barrens[:0]
	tracker.sometimes = tracker.sometimes[:0]
	tracker.always = [7]string{}
}
