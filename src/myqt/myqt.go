package main

import (
	"myq"
	"fmt"
	"os"
	"flag"
	"github.com/rakyll/globalconf"
)

func usage() {
	fmt.Printf("%s: Control Liftmaster MyQ enabled doors\n",
		os.Args[0])
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("  help - this message")
	fmt.Println("  list - show all doors")
	fmt.Println("  locations - show all locations")
	fmt.Println("  details <door> - show details for door <door>")
	fmt.Println("  state <door> - return the state (open|closed) of <door>")
	fmt.Println("  open <door> - Open <door>")
	fmt.Println("  close <door> - Close <door>")
	fmt.Println("  listopen - list all open doors")
	fmt.Println("  listclosed - list all closed doors")
}

func main() {
	var m myq.MyQ
	var d myq.Device
	var err error
	var username, password string
	var debug, machine_parsable bool
	var conf *globalconf.GlobalConf

	flag.Usage = usage
	
	flag.StringVar(&username, "user", "", "Username")
	flag.StringVar(&password, "password", "", "Password")
	flag.BoolVar(&debug, "D", false, "Debugging enabled")
	flag.BoolVar(&machine_parsable, "M", false, "Machine parsable output")

	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Println("No command(s)")
		usage()
		os.Exit(0)
	}

	if flag.Arg(0) == "help" {
		usage()
		os.Exit(0)
	}

	// read confg
	if conf, err = globalconf.New("myqt"); err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	conf.ParseAll()
	
	

	if err := m.New(username, password, debug, machine_parsable); err != nil {
		fmt.Printf("Error: %s\n", err);
		os.Exit(1)
	}

	command := flag.Arg(0)
	door := flag.Arg(1)
	switch command {
	case "help": usage()
	case "state": 
		if d, err = m.FindDoorByName(door); err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		}
		m.GetState(d)
	case "details": 
		if d, err = m.FindDoorByName(door); err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		}
		m.DoorDetails(d)
	case "open": 
		if d, err = m.FindDoorByName(door); err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		}
		if err = m.Open(d); err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		}
	case "close": 
		if d, err = m.FindDoorByName(door); err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		}
		if err = m.Close(d); err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		}
	case "list":
		m.ShowDoors()
	case "locations":
		m.ShowLocations()
	case "listopen":
		m.ShowByState("Open")
	case "listclosed":
		m.ShowByState("Closed")
	default:
		fmt.Printf("unknown command '%s'\n", command)
		os.Exit(1)
	}
}
