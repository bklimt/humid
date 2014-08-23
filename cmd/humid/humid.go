package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/bklimt/hue"
	"github.com/bklimt/midi"
	"io/ioutil"
	"log"
	"strconv"
)

var lights hue.Hue

var presets struct {
	Controls map[string]map[string][]string
	Notes    map[string]map[string]hue.PutLightRequest `json:"notes"`
}

// Coalesced operations for a single light in one network request.
type lightOp struct {
	Light string
	Op    hue.PutLightRequest
	Next  *lightOp
}

// A queue of all the light operations to do over the network.
type lightOpQueue struct {
	Head *lightOp
	Tail *lightOp
}

var q lightOpQueue

func (q *lightOpQueue) pushBack(op lightOp) {
	op.Next = nil
	if q.Tail == nil {
		q.Head = &op
		q.Tail = &op
	} else {
		q.Tail.Next = &op
		q.Tail = q.Tail.Next
	}
}

func (q *lightOpQueue) popFront() (lightOp, bool) {
	if q.Head == nil {
		return lightOp{}, false
	} else {
		e := q.Head
		q.Head = q.Head.Next
		if q.Tail == e {
			q.Tail = nil
		}
		return *e, true
	}
}

func processLightRequests(c chan lightOp) {
	for lop := range c {
		log.Printf("Updating light %v with: %v", lop.Light, lop.Op)
		lights.PutLight(lop.Light, &lop.Op)
	}
}

func enqueue(op lightOp) {
	for p := q.Head; p != nil; p = p.Next {
		if op.Light == p.Light {
			// Merge the two ops.
			log.Printf("Merging op into previous unfinished op: %v <- %v", p, op)
			if op.Op.On != nil {
				p.Op.On = op.Op.On
			}
			if op.Op.Hue != nil {
				p.Op.Hue = op.Op.Hue
			}
			if op.Op.Sat != nil {
				p.Op.Sat = op.Op.Sat
			}
			if op.Op.Bri != nil {
				p.Op.Bri = op.Op.Bri
			}
			log.Printf("Result: %v", p)
			return
		}
	}
	// It wasn't in the queue, so add it at the end.
	log.Printf("Enqueuing operation: %v", op)
	q.pushBack(op)
}

func midiOn(note int) {
	if lightMap, ok := presets.Notes[strconv.Itoa(note)]; ok {
		for light, req := range lightMap {
			enqueue(lightOp{light, req, nil})
		}
	}
}

func midiControl(param, value int) {
	t := true
	f := false
	if lightMap, ok := presets.Controls[strconv.Itoa(param)]; ok {
		for light, attrs := range lightMap {
			req := &hue.PutLightRequest{}
			for _, attr := range attrs {
				switch attr {
				case "on":
					req.On = &t
				case "off":
					req.On = &f
				case "bri":
					if value == 0 {
						req.On = &f
					} else {
						bri := value * 2
						req.Bri = &bri
						req.On = &t
					}
				case "sat":
					sat := value * 2
					req.Sat = &sat
				case "hue":
					hue := (value * 2) << 8
					req.Hue = &hue
				}
			}
			enqueue(lightOp{light, *req, nil})
		}
	}
}

func midiEvent(event interface{}) {
	switch event := event.(type) {
	case midi.Controller:
		fmt.Printf("Controller event: %d %d\n", event.Param, event.Value)
		midiControl(event.Param, event.Value)
	case midi.NoteOn:
		fmt.Printf("Note on: %d\n", event.Note)
		midiOn(event.Note)
	case midi.NoteOff:
		fmt.Printf("Note off: %d\n", event.Note)
	}
}

func loadPresets(filename string) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("Unable to open presets file: %v\n", err)
		return
	}

	err = json.Unmarshal(b, &presets)
	if err != nil {
		fmt.Printf("Unable to parse presets file: %v\n", err)
		return
	}

	fmt.Printf("Presets: %v", presets)
}

func main() {
	ip := flag.String("ip", "192.168.1.3", "IP Address of Philips Hue hub.")
	userName := flag.String("username", "HueGoRaspberryPiUser", "Username for Hue hub.")
	deviceType := flag.String("device_type", "HueGoRaspberryPi", "Device type for Hue hub.")

	presetsFile := flag.String("presets", "./presets.json", "Presets file to use.")

	flag.Parse()

	lights = hue.Hue{*ip, *userName, *deviceType}

	loadPresets(*presetsFile)

	midiChan := make(chan interface{})
	midi.Listen(midiChan)

	httpChan := make(chan lightOp)
	go processLightRequests(httpChan)

	for {
		if q.Head == nil {
			// If the queue is empty just wait on a midi event and put it in the queue.
			midiEvent(<-midiChan)
		} else {
			// If the queue has stuff, then prepare an op and wait for either.
			op := q.Head
			select {
			case event := <-midiChan:
				midiEvent(event)
			case httpChan <- *op:
				q.popFront()
			}
		}
	}
}
