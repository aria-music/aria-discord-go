package aria

import (
	"fmt"
	"time"
)

func durationString(rawdur float64) (dstr string) {
	dur := time.Duration(rawdur) * time.Second
	seconds := (dur % time.Minute) / time.Second
	minutes := (dur % time.Hour) / time.Minute
	hours := dur / time.Hour

	dstr = fmt.Sprintf("%2d:%02d", minutes, seconds)
	if hours > 0 {
		dstr = fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}

	return
}
