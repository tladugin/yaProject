package main

import (
	"fmt"
	"github.com/tladugin/yaProject.git/internal/repository"
	"runtime"
)

func main() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	memStorage := repository.NewMemStorage()
	memStorage.AddGauge("Alloc:", float64(m.Alloc))

	fmt.Println(memStorage.GaugeSlice())
}
