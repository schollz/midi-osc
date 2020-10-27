package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/hypebeast/go-osc/osc"
	osclib "github.com/hypebeast/go-osc/osc"
	log "github.com/schollz/logger"
	driver "gitlab.com/gomidi/rtmididrv"
)

type Config struct {
	Server string  `json:"server"`
	Port   int     `json:"port"`
	Events []Event `json:"events"`
}

type Event struct {
	Comment   string     `json:"comment"`
	Midi      int        `json:"midi"`
	MidiAdd   int        `json:"midi_add,omitempty"`
	Count     int        `json:"count,omitempty"`
	Button    bool       `json:"button,omitempty"`
	OSC       []EventOSC `json:"osc"`
	lastState float32
}

type EventOSC struct {
	Msg     string    `json:"msg"`
	Int32   bool      `json:"int32,omitempty"`
	Float32 bool      `json:"float32,omitempty"`
	Data    float32   `json:"data,omitempty"`   // data to be sent
	Bounds  []float32 `json:"bounds,omitempty"` // midi data bound and sent
	Toggle  []float32 `json:"toggle, omitempty"`
}

func main() {
	log.SetLevel("trace")

	b, err := ioutil.ReadFile("oooooo-nanokontrol.json")
	if err != nil {
		log.Error(err)
		return
	}

	var config Config
	err = json.Unmarshal(b, &config)
	if err != nil {
		log.Error(err)
		return
	}

	events := []Event{}
	for _, e := range config.Events {
		if e.Count > 0 {
			for j := 0; j < e.Count; j++ {
				newe := Event{
					Comment: fmt.Sprintf("%s%d", e.Comment, j+1),
					Midi:    e.Midi + j*e.MidiAdd,
					Button:  e.Button,
					OSC:     []EventOSC{},
				}
				for i, osc := range e.OSC {
					newe.OSC = append(newe.OSC, osc)
					newe.OSC[i].Msg = strings.Replace(newe.OSC[i].Msg, "X", fmt.Sprint(j+1), 1)
				}
				events = append(events, newe)
			}
		} else {
			events = append(events, e)
		}
	}
	log.Debugf("events: %+v", events)
	config.Events = make([]Event, len(events))
	for i, e := range events {
		config.Events[i] = e
	}

	drv, err := driver.New()
	if err != nil {
		log.Error(err)
		return
	}

	ins, err := drv.Ins()
	log.Debugf("ins: %+v", ins)
	if err != nil {
		log.Error(err)
		return
	}

	log.Trace("opening client")
	client := osc.NewClient(config.Server, config.Port)
	limiter := time.Now()
	log.Trace("client opened")
	for i := range ins {
		err = ins[i].Open()
		if err != nil {
			log.Error(err)
			continue
		}
		func(j int) {
			name := ins[j].String()
			log.Tracef("setting up %s", name)
			ins[j].SetListener(func(data []byte, deltaMicroseconds int64) {
				if len(data) == 3 {
					normValue := float32(data[2]) / 127
					midi := int(data[1])
					log.Tracef("midi: %+v", midi)
					if time.Since(limiter) < 50*time.Millisecond {
						return
					}
					for ie, e := range config.Events {
						finished := func(e Event, midi int, val float32) bool {
							if e.Midi != midi {
								return false
							}
							if e.Button && normValue == 0 {
								return true
							}
							for _, osc := range e.OSC {
								if len(osc.Bounds) == 2 {
									val = val * (osc.Bounds[1] - osc.Bounds[0])
									val = val + osc.Bounds[0]
								} else if len(osc.Toggle) == 2 {
									if config.Events[ie].lastState == osc.Toggle[0] {
										val = osc.Toggle[1]
									} else {
										val = osc.Toggle[0]
									}
									config.Events[ie].lastState = val
								} else {
									val = osc.Data
								}
								// send osc data
								msg := osclib.NewMessage(osc.Msg)
								if osc.Int32 {
									msg.Append(int32(val))
								} else {
									msg.Append(float32(val))
								}
								log.Tracef("msg: %+v", msg)
								if time.Since(limiter) > 50*time.Millisecond {
									client.Send(msg)
									time.Sleep(500 * time.Millisecond)
									limiter = time.Now()
								}
							}
							return true
						}(e, midi, normValue)
						if finished {
							break
						}
					}
				}
			})
		}(i)

	}

	c := make(chan bool)
	<-c
}
