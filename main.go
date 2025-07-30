package main

import (
	"flag"
	"log"
	"time"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
)

var (
	targetVolume float64
	interval     time.Duration
)

func init() {
	flag.Float64Var(&targetVolume, "volume", 95.0, "Target microphone volume level (0.00-100.00)")
	flag.DurationVar(&interval, "interval", 3*time.Second, "Interval to check and enforce volume")
	flag.Parse()
}

func main() {
	log.Println("Starting microphone volume enforcer")
	mc, err := NewMicController(float32(targetVolume))
	if err != nil {
		log.Fatal(err)
	}
	defer mc.Close()

	for {
		if err := mc.EnforceVolume(); err != nil {
			log.Println("Failed to enforce volume:", err)
		}
		time.Sleep(interval)
	}
}

type MicController struct {
	targetVolume float32
	volume       *wca.IAudioEndpointVolume
	device       *wca.IMMDevice
	enum         *wca.IMMDeviceEnumerator
}

func NewMicController(targetVolume float32) (*MicController, error) {
	log.Println("Initializing COM")
	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		return nil, err
	}

	log.Println("Creating IMMDeviceEnumerator")
	var enum *wca.IMMDeviceEnumerator
	if err := wca.CoCreateInstance(
		wca.CLSID_MMDeviceEnumerator,
		0,
		wca.CLSCTX_ALL,
		wca.IID_IMMDeviceEnumerator,
		&enum,
	); err != nil {
		ole.CoUninitialize()
		return nil, err
	}

	log.Println("Getting default capture endpoint")
	var device *wca.IMMDevice
	if err := enum.GetDefaultAudioEndpoint(wca.ECapture, wca.EConsole, &device); err != nil {
		enum.Release()
		ole.CoUninitialize()
		return nil, err
	}

	log.Println("Activating IAudioEndpointVolume")
	var volume *wca.IAudioEndpointVolume
	if err := device.Activate(
		wca.IID_IAudioEndpointVolume,
		wca.CLSCTX_ALL,
		0,
		&volume,
	); err != nil {
		device.Release()
		enum.Release()
		ole.CoUninitialize()
		return nil, err
	}

	log.Println("MicController initialized")
	return &MicController{
		targetVolume: targetVolume / 100,
		volume:       volume,
		device:       device,
		enum:         enum,
	}, nil
}

func (m *MicController) EnforceVolume() error {
	var current float32
	if err := m.volume.GetMasterVolumeLevelScalar(&current); err != nil {
		return err
	}
	if current != m.targetVolume {
		log.Printf("Volume was %.2f%%, setting to %.2f%%", current*100, m.targetVolume*100)
		return m.volume.SetMasterVolumeLevelScalar(m.targetVolume, nil)
	}
	log.Printf("Volume is already at %.2f%%, no change needed", current*100)
	return nil
}

func (m *MicController) Close() {
	log.Println("Releasing COM resources")
	m.volume.Release()
	m.device.Release()
	m.enum.Release()
	ole.CoUninitialize()
}
