package mj

func QiDui(sorted_mj []byte) bool {
	if len(sorted_mj) != 14 {
		return false
	}
	for i := 0; i < 14; i += 2 {
		if sorted_mj[i] != sorted_mj[i+1] {
			return false
		}
	}
	return true
}

func PiHu(sorted_mj []byte) bool {
	if len(sorted_mj)%3 != 2 {
		return false
	}
	mj := New(sorted_mj)
	return mj.Pihu()
}
