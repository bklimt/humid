package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/bklimt/hue"
	"github.com/bklimt/midi"
	"io/ioutil"
	"strconv"
)

type PresetMap struct {
	Controls map[string]map[string][]string
	Notes    map[string]map[string]hue.LightRequestBody `json:"notes"`
}

func midiOn(note int, lights *hue.Hue, presets *PresetMap) {
	if lightMap, ok := presets.Notes[strconv.Itoa(note)]; ok {
		for light, req := range lightMap {
			lights.ChangeLight(light, &req)
		}
	}
}

func midiControl(param, value int, lights *hue.Hue, presets *PresetMap) {
	if lightMap, ok := presets.Controls[strconv.Itoa(param)]; ok {
		for light, attrs := range lightMap {
			req := &hue.LightRequestBody{}
			for _, attr := range attrs {
				switch attr {
				case "on":
					t := true
					req.On = &t
				case "off":
					f := false
					req.On = &f
				case "bri":
					bri := value * 2
					req.Bri = &bri
				case "sat":
					sat := value * 2
					req.Sat = &sat
				case "hue":
					hue := (value * 2) << 8
					req.Hue = &hue
				}
			}
			lights.ChangeLight(light, req)
		}
	}
}

func main() {
	ip := flag.String("ip", "192.168.1.3", "IP Address of Philips Hue hub.")
	userName := flag.String("username", "HueGoRaspberryPiUser", "Username for Hue hub.")
	deviceType := flag.String("device_type", "HueGoRaspberryPi", "Device type for Hue hub.")

	presetsFile := flag.String("presets", "./presets.json", "Presets file to use.")

	flag.Parse()

	lights := &hue.Hue{*ip, *userName, *deviceType}

	b, err := ioutil.ReadFile(*presetsFile)
	if err != nil {
		fmt.Printf("Unable to open presets file: %v\n", err)
		return
	}

	var presets PresetMap
	err = json.Unmarshal(b, &presets)
	if err != nil {
		fmt.Printf("Unable to parse presets file: %v\n", err)
		return
	}

	fmt.Printf("Presets: %v", presets)

	c := make(chan interface{})
	midi.Listen(c)
	for event := range c {
		switch event := event.(type) {
		case midi.Controller:
			fmt.Printf("Controller event: %d %d\n", event.Param, event.Value)
			midiControl(event.Param, event.Value, lights, &presets)
		case midi.NoteOn:
			fmt.Printf("Note on: %d\n", event.Note)
			midiOn(event.Note, lights, &presets)
		case midi.NoteOff:
			fmt.Printf("Note off: %d\n", event.Note)
		}
	}
}
