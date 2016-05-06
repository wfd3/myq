# myq

A library and tool in Go to control LiftMaster MyQ-enabled openers.  Reverse engineered from the LiftMaster Web app.  

##PACKAGE DOCUMENTATION

package myq
    import "myq"


TYPES

    type MyQ struct {
        // contains filtered or unexported fields
    }

    func (m *MyQ) Close(d Device) error
Close a specific door/device

    func (m *MyQ) DoorDetails(d Device)
Show details & current state for a specific door/device

    func (m *MyQ) FindDoorByName(name string) (d Device, err error)
Find a device/door by its name

    func (m *MyQ) GetState(d Device)
Show the state (open, closed, ... ) of a specfic door/device

    func (m *MyQ) New(username string, password string, debug bool, machineReadable bool) (err error)
Create a new MyQ Session

    func (m *MyQ) Open(d Device) error
Open a specific door/device

    func (m *MyQ) ShowByState(state string)
Show devices currently in state /state/ (Open, Closed, ... )

    func (m *MyQ) ShowDoors()
Show all devices/doors

    func (m *MyQ) ShowLocations()
Show all locations associated with this MyQ account

##MyQ TOOL 
    myqt [-DM] [-user username] [-password password] <command>
    
    Options:
      -D=false: Debugging enabled
      -M=false: Machine parsable output
      -password="": Password
      -user="": Username
    
    Commands:  
      help - this message
      list - show all doors
      locations - show all locations
      details <door> - show details for door <door>
      state <door> - return the state (open|closed) of <door>
      open <door> - Open <door>
      close <door> - Close <door>
      listopen - list all open doors
      listclosed - list all closed doors

