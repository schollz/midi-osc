package main

import (
	"github.com/hypebeast/go-osc/osc"
	log "github.com/schollz/logger"
	driver "gitlab.com/gomidi/rtmididrv"
)

func main() {
	drv, err := driver.New()
	if err != nil {
		return
	}

	ins, err := drv.Ins()
	if err != nil {
		return
	}

	client := osc.NewClient("192.168.0.82", 10111)
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
					log.Tracef("[%s] %+v", name, data)
					if data[1] == 0 {
						msg := osc.NewMessage("/param/1vol")
						msg.Append(float32(data[2]) / 127)
						client.Send(msg)
					}
					if data[1] == 38 && data[2] == 127 {
						log.Info("turning on compressor")
						msg := osc.NewMessage("/param/compressor")
						msg.Append(int32(127))
						client.Send(msg)
						msg = osc.NewMessage("/param/comp_mix")
						msg.Append(float32(data[2]) / 127)
						client.Send(msg)
					}
					if data[1] == 54 && data[2] == 127 {
						log.Info("turning off compressor")
						msg := osc.NewMessage("/param/compressor")
						msg.Append(int32(0))
						client.Send(msg)
					}
					if data[1] == 39 && data[2] == 127 {
						log.Info("turning on reverb")
						msg := osc.NewMessage("/param/reverb")
						msg.Append(int32(127))
						client.Send(msg)
						msg = osc.NewMessage("/param/rev_monitor_input")
						msg.Append(float32(-9.0))
						client.Send(msg)
						msg = osc.NewMessage("/param/rev_return_level")
						msg.Append(float32(6))
						client.Send(msg)
					}
					if data[1] == 55 && data[2] == 127 {
						log.Info("turning off compressor")
						msg := osc.NewMessage("/param/reverb")
						msg.Append(int32(0))
						client.Send(msg)
					}
				}
			})
		}(i)

	}

	c := make(chan bool)
	<-c
}
