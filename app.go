package main

import (
	"errors"
	"ivan/timer"
	"ivan/tracker"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/inpututil"
)

const configPath = "assets/config.json"

var errCloseApp = errors.New("user requested app close")

type App struct {
	tracker *tracker.Tracker
	timer   *timer.Timer
	config  config
}

func NewApp() (*App, error) {
	config, err := loadConfig(configPath)
	if err != nil {
		return nil, err
	}

	size := config.windowSize()
	ebiten.SetWindowSize(size.X, size.Y)
	ebiten.SetWindowPosition(1920-size.X, 0)

	timer, err := timer.New(config.Dimensions.Timer)
	if err != nil {
		return nil, err
	}

	tracker, err := tracker.New(
		config.Dimensions.ItemTracker,
		config.Dimensions.HintTracker,
		config.Items,
		config.ZoneItemMap,
		config.Locations,
	)
	if err != nil {
		return nil, err
	}

	return &App{
		tracker: tracker,
		timer:   timer,
		config:  config,
	}, nil
}

func (app *App) Update(screen *ebiten.Image) error {
	_, wheel := ebiten.Wheel()

	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		if !app.timer.IsRunning() && !app.tracker.EatInput() {
			return errCloseApp
		}
		app.tracker.Cancel()

	case inpututil.IsKeyJustPressed(ebiten.KeyHome):
		if !app.timer.IsRunning() {
			config, err := loadConfig(configPath)
			if err != nil {
				return err
			}
			app.config = config
			app.tracker.Reset(app.config.Items, app.config.ZoneItemMap)
		}

	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		app.tracker.Submit()

	case inpututil.IsKeyJustPressed(ebiten.KeySpace):
		if app.tracker.EatInput() {
			app.tracker.Input([]rune(" "))
		} else {
			app.timer.Toggle()
		}

	case inpututil.IsKeyJustPressed(ebiten.KeyDelete):
		app.timer.Reset()

	case inpututil.IsKeyJustPressed(ebiten.KeyBackspace):
		app.tracker.Backspace()

	case inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft):
		app.tracker.ClickLeft(ebiten.CursorPosition())

	case inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight):
		app.tracker.ClickRight(ebiten.CursorPosition())

	case wheel != 0:
		x, y := ebiten.CursorPosition()
		app.tracker.Wheel(x, y, wheel > 0)

	default:
		app.tracker.Input(ebiten.InputChars())
	}

	return nil
}

func (app *App) Draw(screen *ebiten.Image) {
	app.tracker.Draw(screen)
	app.timer.Draw(screen)
}

func (app *App) Layout(w, h int) (int, int) {
	return w, h
}
