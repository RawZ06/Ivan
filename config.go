package main

import (
	"encoding/json"
	"image"
	"ivan/tracker"
	"os"
)

type config struct {
	Items       []tracker.Item
	ZoneItemMap [9][9]string
	Locations   []string // woth/barren "simple" locations
	Dimensions  struct {
		ItemTracker image.Rectangle
		Timer       image.Rectangle
		HintTracker image.Rectangle
	}
}

func (c config) windowSize() image.Point {
	var ret image.Rectangle
	for _, v := range []image.Rectangle{
		c.Dimensions.ItemTracker,
		c.Dimensions.Timer,
		c.Dimensions.HintTracker,
	} {
		ret = ret.Union(v)
	}

	return ret.Size()
}

func loadConfig(path string) (config, error) {
	f, err := os.Open(path)
	if err != nil {
		return config{}, err
	}
	defer f.Close()

	var ret config
	dec := json.NewDecoder(f)
	if err := dec.Decode(&ret); err != nil {
		return config{}, err
	}

	return ret, nil
}
