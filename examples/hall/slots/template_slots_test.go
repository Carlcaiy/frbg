package slots

import (
	"frbg/def"
	"log"
	"testing"
)

func TestGet(t *testing.T) {
	slots := GetSlotsData(1, def.SlotsFu)
	// for i := 0; i < 100; i++ {
	// 	fmt.Println(slots.Rand.Int31n(100))
	// }
	log.Printf("%+v\n", slots)
	slots.Spin(10)
	log.Printf("%+v\n", slots)
}
