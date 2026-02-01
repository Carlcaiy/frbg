package main

import (
	"fmt"
	"frbg/mj"
)

func main() {
	st := mj.Newlz([]uint8{11, 11, 12, 12, 12, 12, 13, 13}, 0, 13, nil)
	fmt.Println(mj.HuStr(st.HuPai()))
}
