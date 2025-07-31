package main

import (
	"flag"
	"log"
	"sync/atomic"
	"time"

	"github.com/morgahl/force_volume/internal/httpserver"
	"github.com/morgahl/force_volume/internal/mic"
)

var (
	targetVolume float64
	interval     time.Duration
	htmlAddr     string
)

var intervalMillis atomic.Int64
var volumePercent atomic.Int64

func init() {
	flag.Float64Var(&targetVolume, "volume", 95.0, "Target microphone volume level (0.00-100.00)")
	flag.DurationVar(&interval, "interval", 3*time.Second, "Interval to check and enforce volume")
	flag.StringVar(&htmlAddr, "html", "", "Optional address:port to serve HTML control UI")
	flag.Parse()

	if targetVolume < 0 || targetVolume > 100 {
		log.Fatal("Volume must be between 0.00 and 100.00")
	}
	if interval < time.Millisecond {
		log.Fatal("Interval must be at least 1 millisecond")
	}
	intervalMillis.Store(interval.Milliseconds())
	volumePercent.Store(int64(targetVolume))
}

func main() {
	log.Println("Starting microphone volume controller")
	mc, err := mic.NewMicController(float32(targetVolume))
	if err != nil {
		log.Fatal(err)
	}
	defer mc.Close()

	if htmlAddr != "" {
		go httpserver.Start(htmlAddr, &intervalMillis, &volumePercent)
	}

	for {
		mc.SetTargetVolume(float32(volumePercent.Load()) / 100)
		if err := mc.EnforceVolume(); err != nil {
			log.Println("Failed to enforce volume:", err)
		}
		time.Sleep(time.Duration(intervalMillis.Load()) * time.Millisecond)
	}
}
