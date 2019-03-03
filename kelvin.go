// MIT License
//
// Copyright (c) 2019 Stefan Wichmann
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
)

var applicationVersion = "development"
var debug = flag.Bool("debug", false, "Enable debug logging")
var logFile = flag.String("log", "", "Redirect log output to specified file")
var configurationFile = flag.String("configuration", absolutePath("config.json"), "Specify the filename of the configuration to load")
var forceUpdate = flag.Bool("forceUpdate", false, "Update to new major version")
var enableWebInterface = flag.Bool("enableWebInterface", false, "Enable the web interface at startup")

var configuration *Configuration
var bridge = &HueBridge{}
var lights []*Light

const lightUpdateIntervalInSeconds = 1
const stateUpdateIntervalInSeconds = 60

func main() {
	flag.Parse()
	configureLogging()
	log.Printf("Kelvin %v starting up... 🚀", applicationVersion)
	log.Debugf("Current working directory: %v", workingDirectory())
	go CheckForUpdate(applicationVersion, *forceUpdate)
	go validateSystemTime()
	go handleSIGHUP()

	// load configuration or create a new one
	conf, err := InitializeConfiguration(*configurationFile, *enableWebInterface)
	if err != nil {
		log.Fatal(err)
	}
	configuration = &conf

	// start interface
	go startInterface()

	// find bridge
	err = bridge.InitializeBridge(configuration)
	if err != nil {
		log.Warning(err)
	}

	// find location
	_, err = InitializeLocation(configuration)
	if err != nil {
		log.Warning(err)
	}

	// save configuration
	err = configuration.Write()
	if err != nil {
		log.Fatal(err)
	}

	// start routine for the scenes
	go updateScenesCyclic()

	// initialize all lights
	lights, err = bridge.Lights()
	if err != nil {
		log.Warning(err)
	}

	// implicit schedule, target and interval
	for _, light := range lights {
		light := light
		updateScheduleForLight(light)
	}

	// start cyclic update for all lights
	log.Debugf("💡 Starting cyclic update...")
	lightUpdateTick := time.Tick(lightUpdateIntervalInSeconds * time.Second)
	stateUpdateTick := time.Tick(stateUpdateIntervalInSeconds * time.Second)
	for {
		select {
		case <-time.After(durationUntilEndOfDay()):
			// Day has ended, calculate new schedule
			for _, light := range lights {
				light := light
				updateScheduleForLight(light)
			}
		case <-stateUpdateTick:
			// update interval and color every minute
			for _, light := range lights {
				light := light
				light.updateInterval()
				light.updateTargetLightState()
			}
		case <-lightUpdateTick:
			states, err := bridge.LightStates()
			if err != nil {
				log.Warning(err)
			}
			for _, light := range lights {
				light := light
				currentLightState, found := states[light.ID]
				if found {
					light.updateCurrentLightState(currentLightState)
					light.update()
				} else {
					log.Warningf("No current light state found for light %d", light.ID)
				}
			}
		}
	}
}

func updateScheduleForLight(light *Light) {
	schedule, err := configuration.lightScheduleForDay(light.ID, time.Now())
	if err != nil {
		log.Printf("💡 Light %s - Light is not associated to any schedule. Ignoring...", light.Name)
		light.Schedule = schedule // Assign empty schedule
		light.Scheduled = false
	} else {
		light.updateSchedule(schedule)
		light.updateInterval()
		light.updateTargetLightState()
	}
}
func handleSIGHUP() {
	sighup := make(chan os.Signal, 1)
	signal.Notify(sighup, syscall.SIGHUP)
	<-sighup // wait for signal
	log.Printf("Received signal SIGHUP. Restarting...")
	Restart()
}

func configureLogging() {
	formatter := new(log.TextFormatter)
	formatter.FullTimestamp = true
	formatter.TimestampFormat = "2006/02/01 15:04:05"
	log.SetFormatter(formatter)
	if *debug {
		log.SetLevel(log.DebugLevel)
	}
	if logFile != nil && *logFile != "" {
		file, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			log.SetOutput(file)
		} else {
			log.Info("Failed to log to file, using default stderr")
		}
	}
}

func validateSystemTime() {
	// validate local clock as it forms the basis for all time calculations.
	valid, err := IsLocalTimeValid()
	if err != nil {
		log.Fatal(err)
	}
	if !valid {
		log.Warningf("WARNING: Your local system time seems to be more than one minute off. Timings may be inaccurate.")
	} else {
		log.Debugf("Local system time validated.")
	}
}
