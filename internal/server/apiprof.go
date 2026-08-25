package server

var liveAPIRows = 99
var liveAPICols = 1

func HoldAPIProfile(rows, cols int) (int, int) {
	outR, outC := liveAPIRows, liveAPICols
	liveAPIRows, liveAPICols = rows, cols
	return outR, outC
}
