package network

import "frbg/def"

func Signal(polls ...*Poll) {
	for _, poll := range polls {
		poll.Trigger(def.ET_Close)
	}
}

func Wait(polls ...*Poll) {
	wg.Wait()
	for _, poll := range polls {
		poll.Close()
	}
}
