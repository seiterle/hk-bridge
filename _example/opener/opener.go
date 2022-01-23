package main

// Door Opener for an doorbell with an electric door opener.

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/service"

	"github.com/seiterle/hr/bridge"
	"github.com/stianeikeland/go-rpio/v4"
)

const (
	relayOne = 23
)

type Lock struct {
	*accessory.Accessory

	Lock *service.LockMechanism
}

func NewLock(info accessory.Info) *Lock {

	// accessory
	acc := Lock{}
	acc.Accessory = accessory.New(info, accessory.TypeDoorLock)
	acc.Lock = service.NewLockMechanism()
	acc.AddService(acc.Lock.Service)

	return &acc
}

func main() {

	ac := NewLock(accessory.Info{
		Name: "Main Door",
	})

	if err := rpio.Open(); err != nil {
		log.Fatal("Failed to access GPIO interface: %v", err)
	}

	relayOpener := rpio.Pin(relayOne)
	relayOpener.Output()
	relayOpener.Low()
	defer relayOpener.Low()
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		relayOpener.Low()
		os.Exit(0)
	}()

	// This is an electric strike lock that is only ever open for 1 second
	go func() {
		for {
			if characteristic.LockCurrentStateSecured != ac.Lock.LockCurrentState.GetValue() {
				relayOpener.Low()
				ac.Lock.LockTargetState.SetValue(characteristic.LockTargetStateSecured)
				ac.Lock.LockCurrentState.SetValue(characteristic.LockCurrentStateSecured)
			}
			if characteristic.LockTargetStateUnsecured == ac.Lock.LockTargetState.GetValue() {
				relayOpener.High()
				ac.Lock.LockCurrentState.SetValue(characteristic.LockCurrentStateUnsecured)
				time.Sleep(2 * time.Second)
			}
			time.Sleep(time.Millisecond * 250)
		}
	}()

	b, err := bridge.NewBridge(ac.Accessory)
	if err != nil {
		log.Fatal("failed to create bridge")
	}
	b.Start()
}
