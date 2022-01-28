package main

// Door Opener for an doorbell with an electric door opener.

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/service"
	"github.com/stianeikeland/go-rpio/v4"

	"github.com/seiterle/hr/bridge"
)

const (
	gpioLock = 23
	gpioBell = 12

	numberStateSwitches = 4
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

type Doorbell struct {
	*accessory.Accessory

	Control           *service.Doorbell
	StreamManagement1 *service.CameraRTPStreamManagement
	Speaker           *service.Speaker
	Microphone        *service.Microphone
}

func NewDoorbell(info accessory.Info) *Doorbell {
	acc := Doorbell{}
	acc.Accessory = accessory.New(info, accessory.TypeVideoDoorbell)
	acc.Control = service.NewDoorbell()
	acc.AddService(acc.Control.Service)

	acc.StreamManagement1 = service.NewCameraRTPStreamManagement()
	acc.AddService(acc.StreamManagement1.Service)

	acc.Speaker = service.NewSpeaker()
	acc.AddService(acc.Speaker.Service)

	acc.Microphone = service.NewMicrophone()
	acc.AddService(acc.Microphone.Service)

	return &acc
}

type DoorbellButton struct {
	*accessory.Accessory

	ProgrammableSwitch *service.StatelessProgrammableSwitch
}

func NewDoorbellButton(info accessory.Info) *DoorbellButton {
	acc := DoorbellButton{}
	acc.Accessory = accessory.New(info, accessory.TypeProgrammableSwitch)
	acc.ProgrammableSwitch = service.NewStatelessProgrammableSwitch()
	acc.ProgrammableSwitch.ProgrammableSwitchEvent.SetMaxValue(characteristic.ProgrammableSwitchEventSinglePress)
	acc.AddService(acc.ProgrammableSwitch.Service)
	return &acc
}

func main() {

	as := []*accessory.Accessory{}

	if err := rpio.Open(); err != nil {
		log.Fatal("Failed to access GPIO interface: %v", err)
	}

	pinLock := rpio.Pin(gpioLock)
	pinLock.Output()
	pinLock.Low()
	defer pinLock.Low()
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		pinLock.Low()
		os.Exit(0)
	}()

	pinBell := rpio.Pin(gpioBell)
	pinBell.Input()

	lock := NewLock(accessory.Info{
		Name: "Lock",
	})

	// This is an electric strike lock that is only ever open for 1 second
	go func() {
		for {
			if characteristic.LockCurrentStateSecured != lock.Lock.LockCurrentState.GetValue() {
				pinLock.Low()
				lock.Lock.LockTargetState.SetValue(characteristic.LockTargetStateSecured)
				lock.Lock.LockCurrentState.SetValue(characteristic.LockCurrentStateSecured)
			}
			if characteristic.LockTargetStateUnsecured == lock.Lock.LockTargetState.GetValue() {
				pinLock.High()
				lock.Lock.LockCurrentState.SetValue(characteristic.LockCurrentStateUnsecured)
				time.Sleep(2 * time.Second)
			}
			time.Sleep(time.Millisecond * 250)
		}
	}()

	bell := NewDoorbell(accessory.Info{
		Name: "Doorbell",
	})

	// add the doorbell also as a programmable switch
	bellButton := NewDoorbellButton(accessory.Info{
		Name: "Doorbell",
	})

	go func() {
		for {
			if rpio.Low != pinBell.Read() {
				bell.Control.ProgrammableSwitchEvent.SetValue(characteristic.ProgrammableSwitchEventSinglePress)
				bellButton.ProgrammableSwitch.ProgrammableSwitchEvent.SetValue(characteristic.ProgrammableSwitchEventSinglePress)
				time.Sleep(3 * time.Second)
			}
			time.Sleep(time.Millisecond * 250)
		}
	}()

	as = append(as, lock.Accessory, bell.Accessory, bellButton.Accessory)

	// add dummy switches to store state
	for i := 0; i < numberStateSwitches; i++ {
		s := accessory.NewSwitch(accessory.Info{
			Name: fmt.Sprintf("Switch %v", i),
		})
		as = append(as, s.Accessory)
	}

	b, err := bridge.NewBridge(as...)
	if err != nil {
		log.Fatal("failed to create bridge")
	}
	b.Start()
}
