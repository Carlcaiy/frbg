package mj

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestHu(t *testing.T) {
	mj := []byte{
		1, 2, 3, 4, 5, 6, 7,
		1, 2, 3, 4, 5, 6, 7,
		1, 2, 3, 4, 5, 6, 7,
		1, 2, 3, 4, 5, 6, 7,
		11, 12, 13, 14, 15, 16, 17, 18, 19,
		11, 12, 13, 14, 15, 16, 17, 18, 19,
		11, 12, 13, 14, 15, 16, 17, 18, 19,
		11, 12, 13, 14, 15, 16, 17, 18, 19,
		21, 22, 23, 24, 25, 26, 27, 28, 29,
		21, 22, 23, 24, 25, 26, 27, 28, 29,
		21, 22, 23, 24, 25, 26, 27, 28, 29,
		21, 22, 23, 24, 25, 26, 27, 28, 29,
		31, 32, 33, 34, 35, 36, 37, 38, 39,
		31, 32, 33, 34, 35, 36, 37, 38, 39,
		31, 32, 33, 34, 35, 36, 37, 38, 39,
		31, 32, 33, 34, 35, 36, 37, 38, 39,
	}
	for i := 0; i < 1000000; i++ {
		rand.Shuffle(len(mj), func(i, j int) {
			mj[i], mj[j] = mj[j], mj[i]
		})
		hands1 := mj[:14]
		hands2 := mj[14:28]
		hands3 := mj[28:42]
		hands4 := mj[42:56]
		if st := New(hands1); st.Pihu() {
			fmt.Println(st, hands1)
		}
		if st := New(hands2); st.Pihu() {
			fmt.Println(st, hands2)
		}
		if st := New(hands3); st.Pihu() {
			fmt.Println(st, hands3)
		}
		if st := New(hands4); st.Pihu() {
			fmt.Println(st, hands4)
		}
	}
}

func TestMj(t *testing.T) {
	mj := []byte{3, 7, 12, 17, 19, 19, 22, 23, 24, 25, 29, 29, 37, 37}
	// st := &Mj{
	// 	Val: []byte{1, 12, 13, 14, 15, 16, 19, 21, 33, 35, 36},
	// 	Num: []byte{2, 2, 1, 1, 1, 1, 1, 1, 1, 2, 1},
	// }
	fmt.Println(New(mj).Pihu())
}
