package main

//go:generate go run internal/generate/ymlGenerator.go

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/Cloud-Pie/Passa/cloudsolution"
	"github.com/Cloud-Pie/Passa/cloudsolution/dockerswarm"
	"github.com/Cloud-Pie/Passa/notification"
	"github.com/Cloud-Pie/Passa/notification/consoleprinter"
	"github.com/Cloud-Pie/Passa/notification/telegram"
	"github.com/Cloud-Pie/Passa/server"
	"github.com/Cloud-Pie/Passa/ymlparser"
)

const (
	defaultLogFile = "test.log"
	defaultYMLFile = "test/passa-states-test.yml"
)

var notifier notification.NotifierInterface
var flagVars flagVariable

func main() {

	var err error
	flagVars = parseFlags()
	c := ymlparser.ParseStatesfile(flagVars.configFile)

	//Notifier code Start
	notifier, err = telegram.InitializeClient()

	if err != nil {
		notifier = consoleprinter.InitializeClient()
	}
	//Notifier code End

	//Code For Cloud Management Start

	var cloudManager cloudsolution.CloudManagerInterface
	if !flagVars.noCloud {
		if c.Provider.Name == "docker-swarm" {
			cloudManager = dockerswarm.NewSwarmManager(c.Provider.ManagerIP)
		}
	}

	for idx := range c.States {

		state := &c.States[idx]
		durationUntilStateChange := state.ISODate.Sub(time.Now())
		deploymentTimer := time.AfterFunc(durationUntilStateChange, scale(cloudManager, *state)) //Golang closures
		state.SetTimer(deploymentTimer)
		fmt.Printf("Deployment: %v\n", state)

	}
	//Code For Cloud Management End

	//Server code Start
	server := server.SetupServer(c) //BUG: add channel for state -> scaler comm.
	server.Run()
	//Server code End
}

func scale(manager cloudsolution.CloudManagerInterface, s ymlparser.State) func() {

	return func() {

		if !flagVars.noCloud {
			manager = manager.ChangeState(s)
			fmt.Printf("%#v", manager.GetLastDeployedState())
		}
		notifier.Notify("Deployed " + s.Name)

	}
}

func setLogFile(lf string) string {
	if lf == "" {
		lf = defaultLogFile
	}
	fmt.Println("Writing log to  -> ", lf)
	os.MkdirAll(filepath.Dir(lf), 0700)
	f, err := os.OpenFile(lf, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	log.SetOutput(f)
	return lf
}

type flagVariable struct {
	noCloud    bool
	configFile string
	logFile    string
}

func parseFlags() flagVariable {
	noCloud := flag.Bool("no-cloud", false, "Don't start cloud management") //NOTE: For testing only
	configFile := flag.String("state-file", defaultYMLFile, "config file")
	logFile := flag.String("test-file", defaultLogFile, "log file")

	flag.Parse()
	return flagVariable{
		noCloud:    *noCloud,
		configFile: *configFile,
		logFile:    *logFile,
	}
}
