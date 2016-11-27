package myq

import (
	"encoding/json"
	"errors"
	"strconv"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
	"os"
	"net/http/cookiejar"
)

var _Culture string = "en-US"	
var _BaseURL = "https://www.myliftmaster.com/"
var _BrandName = "LiftMaster"

// Desired state (ie, I want the door to open)
const (
	desiredState_Closed = 0
	desiredState_Open   = 1
)

type devices []Device
type Device struct {
	// JSON representation
	Gatewayid int `json:"GatewayId"`
	Errorstatus string `json:"ErrorStatus"`
	Errormessage string `json:"ErrorMessage"`
	Lastupdateddatetime time.Time `json:"LastUpdatedDateTime"`
	Gateway string `json:"Gateway"`
	Myqdeviceid int `json:"MyQDeviceId"`
	Imagesource string `json:"Imagesource"`
	Statesince int64 `json:"Statesince"`
	Displaystatesince string `json:"DisplayStatesince"`
	Name string `json:"Name"`
	State string `json:"State"`
	Error bool `json:"Error"`
	Connectserverdeviceid string `json:"ConnectServerDeviceId"`
	Monitoronly bool `json:"MonitorOnly"`
	Lowbattery bool `json:"LowBattery"`
	Sensorerror bool `json:"SensorError"`
	Openerror bool `json:"OpenError"`
	Closeerror bool `json:"CloseError"`
	Disablecontrol bool `json:"DisableControl"`
	Statename string `json:"StateName"`
	Devicetypeid int `json:"DeviceTypeId"`
	Toggleattributename string `json:"ToggleAttributeName"`
	Toggleattributevalue string `json:"ToggleAttributeValue"`
	//  local additions
	location string
}

type triggerStateChangeReturn struct {
	Errormessage string `json:"errormessage"`
}

// There's something odd about this JSON structure. I don't understand
// why PlacesList needs to be a structure, rather than just a type
// []Places.  As a workaround, I've defined two structures, the actual
// JSON->Go representation, and a "shadow" type (type places) to allow range()
// operations to work as expected.  getAllGateways() converts from one
// to the other.
type placeList struct {		// This is the JSON representation
	P []struct {
		Gatewayid int `json:"GatewayId"`
		Name string `json:"Name"`
		Connectserverid string `json:"ConnectServerId"`
		Devicelist string `json:"DeviceList"`
		Isdetonator bool `json:"IsDetonator"`
	}  `json:"Placeslist"`
}
type places map[int]place // Local mapping of PlacesList
type place struct {
	Gatewayid int `json:"GatewayId"`
	Name string `json:"Name"`
	Connectserverid string `json:"ConnectServerId"`
	Devicelist string `json:"DeviceList"`
	Isdetonator bool `json:"IsDetonator"`
}

type MyQ struct {
	c http.Client
	devices devices
	locations places
	debug bool
	machineReadable bool
}

// Helpers

func (m *MyQ) debugf(format string, a ...interface{}) (n int, err error) {
	if m.debug {
		format = "# " + format
		return fmt.Fprintf(os.Stderr, format, a...)
	}
	return 0, nil
}

// Do a HTTPS GET and parse the JSON response
func (m *MyQ) doGet(rawurl string, v url.Values, s interface{}) (err error) {
	var r []byte
	var res *http.Response

	u, _ := url.Parse(rawurl)
	u.RawQuery = v.Encode()
	m.debugf("doGet():  URL -  %s\n", u.String())

	t := time.Now()
	if res, err = m.c.Get(u.String()); err != nil {
		m.debugf("Get() failed: %s\n", err)
		return err
	}
	d := time.Since(t)
	m.debugf("   HTTP Response: %s, in %s\n", res.Status, d)
	if res.StatusCode != http.StatusOK {
		return errors.New(res.Status)
	}
	r, err = ioutil.ReadAll(res.Body)
	res.Body.Close()

	if err != nil { 
		m.debugf("ReadAll() failed: %s\n", err)
		return err
	}
	return json.Unmarshal(r, &s)
}

// Do a HTTPS POST and unmarshal the JSON result, if we're expecting a result (ie, s != nil)
func (m *MyQ) doPost(rawurl string, v url.Values, s interface{}) (err error) {
	var res *http.Response

	if m.debug {
		u, _ := url.Parse(rawurl)
		u.RawQuery = v.Encode()
		m.debugf("doPost(): URL - %s\n", u)
	}
	t := time.Now()
	if res, err = m.c.PostForm(rawurl, v); err != nil {
		m.debugf("Post() failed: %s\n", err)
		return err
	}
	d := time.Since(t)
	m.debugf("   HTTP Response: %s, in %s\n", res.Status, d)
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP Error: %s", res.Status)
	}

	r, err := ioutil.ReadAll(res.Body)
	res.Body.Close()

	if s != nil {
		return json.Unmarshal(r, &s)
	}
	return nil
}

// Functions to handle type place
func (p *place) print(machinereadable bool) string {
	if machinereadable {
	return fmt.Sprintf("%s,%d", p.Name, p.Gatewayid)
	} 
	return fmt.Sprintf("%s (ID %d)", p.Name, p.Gatewayid)
}

func (p *place) string() string {
	return p.print(false)
}

// Functions to handle type Device
func (d Device) print(machinereadable bool) string {
	var s string
	
	d.Lastupdateddatetime = d.Lastupdateddatetime.Local()
	if machinereadable {
		s = fmt.Sprintf("%s,%s,%d,%s,%d,%s,%s", d.Name, d.location,
			d.Myqdeviceid, d.Statename,
			d.Lastupdateddatetime.Unix(), d.Errorstatus,
			d.Errormessage)
		if d.Monitoronly {
			s += ",Monitor"
		}
		if d.Lowbattery {
			s += ",LowBat"
		}
		if d.Sensorerror {
			s += ",SensorErr"
		}
		if d.Openerror {
			s += ",OpenErr"
		}
		if d.Closeerror {
			s += ",CloseErr"
		}
		if d.Disablecontrol {
			s += ",Disabled"
		}
	} else {
		s = fmt.Sprintf("%s at %s is %s since %v",
			d.Name, d.location, d.Statename,
			d.Lastupdateddatetime.Format(time.Stamp))
		if d.Error {
			s += fmt.Sprintf(", ERROR: status = %s, message = %s", 
				d.Errorstatus, d.Errormessage)
		}
		if d.Monitoronly {
			s += ",Monitor Only"
		}
		if d.Lowbattery {
			s += ", LowBat"
		}
		if d.Sensorerror {
			s += ", Sensor Error"
		}
		if d.Openerror {
			s += ", Open Error"
		}
		if d.Closeerror {
			s += ", Close Error"
		}
		if d.Disablecontrol {
			s += ", Control disabled"
		}
	}
	return s
}

func (d Device) String() string {
	return d.print(false)
}

// MyQ REST API
func (m *MyQ) getAllGateways() (err error) {
	var j placeList

	// The only option is a current timestamp in ms
	v := url.Values{}
	v.Add("_", strconv.FormatInt(time.Now().UnixNano()/1000000, 10))
	if err = m.doGet(_BaseURL + "Gateway/GetAllGateways", v, &j);
	err != nil {
		return err
	}

	// Convert the JSON array to a map
	m.locations = make(places)
	for _, p := range(j.P) {
		m.locations[p.Gatewayid] = p
	}

	return err
}
	
func (m *MyQ) getAllDevices() (err error) {
	var d devices

	if err = m.getAllGateways(); err != nil {
		return err
	}

	v := url.Values{}
	v.Add("culture", _Culture)
	v.Add("brandName", _BrandName)
	if err = m.doGet(_BaseURL + "api/MyQDevices/GetAllDevices", v, &d);
	err != nil {
		return err
	}

	// Fixup the json decoding.  This isn't an issue with the Go
	// JSON parser but rather bugs/implementation details in the
	// MyQ service (from what I can tell).
	for i, x := range(d) {
		// Set the location for each device
		d[i].location = m.locations[x.Gatewayid].Name
		// Fix unknown doorstates
		if d[i].Statename == "" {
			d[i].Statename = "Unknown"
		}
	}

	m.devices = d
	return nil
}

func (m *MyQ) setDoorState(d Device, desiredstate int) (err error) {
	var t triggerStateChangeReturn

	m.debugf("SetDoorState: desiredstate = %d\n", desiredstate)
	if desiredstate != desiredState_Open &&
	   desiredstate != desiredState_Closed {
		return errors.New("Invalid door state")
	}

	v := url.Values{}
	v.Add("myQDeviceId", strconv.Itoa(d.Myqdeviceid))
	v.Add("attributename", "desireddoorstate")
	v.Add("attributevalue", strconv.Itoa(desiredstate))

	err = m.doPost(_BaseURL + "Device/TriggerStateChange", v, t)
	if t.Errormessage != "" {
		return errors.New(t.Errormessage) 
	}
	return err
}

func (m *MyQ) login(username string, password string) (err error) {

	v := url.Values{}
	v.Add("Email", username)
	v.Add("Password", password)
	
	if m.c.Jar, err = cookiejar.New(nil); err != nil {
		return fmt.Errorf("login(): Can't create CookieJar: %s", err)
	}
	m.c.Timeout = 60 * time.Second

	if  err = m.doPost(_BaseURL, v, nil); err != nil {
		return fmt.Errorf("Post() failed: %s\n", err)
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////
// Public API

// Create a new MyQ Session
func (m *MyQ) New(username string, password string, debug bool,
	machineReadable bool) (err error) {
	m.debug = debug
	m.machineReadable = machineReadable 

	if err = m.login(username, password); err != nil {
		m.debugf("login error: %s", err)
		return errors.New("Login failed")
	}
	
	if err = m.getAllDevices(); err != nil {
		m.debugf("getAllDevices() error: %s", err)
		return errors.New("Can't get device list")
	}

	return err
}

// Find a device/door by its name
func (m *MyQ) FindDoorByName(name string) (d Device, err error) {
	for _, d = range m.devices {
		if d.Name == name {
			return d, nil
		}
	}
	return d, fmt.Errorf("Device named '%s' not found", name)
}

// Show all devices/doors
func (m *MyQ) ShowDoors() {
	for _, d := range m.devices {
		fmt.Println(d.print(m.machineReadable))
	}
}

// Show all locations associated with this MyQ account
func (m *MyQ) ShowLocations() {
	for _, x := range m.locations {
		fmt.Println(x.print(m.machineReadable))
	}
}

// Show details & current state for a specific door/device
func (m *MyQ) DoorDetails(d Device){
	fmt.Println(d.print(m.machineReadable))
}

// Show the state (open, closed, ... ) of a specfic door/device
func (m *MyQ) GetState(d Device) {
	fmt.Println(d.Statename)
}

// Open a specific door/device
func (m *MyQ) Open(d Device) error {
	if d.Statename == "Open" {
		return errors.New("Door is already open")
	} else if d.Statename != "Closed" {
		return fmt.Errorf("Can't open, door is currently %s",
			d.Statename)
	}
	return m.setDoorState(d, desiredState_Open)
}	

// Close a specific door/device
func (m *MyQ) Close(d Device) error {
	if d.Statename == "Closed" {
		return errors.New("Door already closed")
	} else if d.Statename != "Open" {
		return fmt.Errorf("Can't close, door is currently %s",
			d.Statename)
	}
	return m.setDoorState(d, desiredState_Closed)
}	

// Show devices currently in state /state/ (Open, Closed, ... )
func (m *MyQ) ShowByState(state string) {
	for _, d := range m.devices {
		if d.Statename == state {
			fmt.Println(d.print(m.machineReadable))
		}
	}
}
