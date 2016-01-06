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
	"golang.org/x/net/html"
	"net/http/cookiejar"
	"golang.org/x/net/html/atom"
)

var _Culture string = "en-US"	
var _BaseURL = "https://www.myliftmaster.com/"
var _BrandName = "LiftMaster"

// Current states (ie, the door is currently closed)
const (
	Doorstate_UNKNOWN = -1
	Doorstate_Open    = 1
	Doorstate_Closed  = 2
	Doorstate_Stopped = 3
	Doorstate_Opening = 4
	Doorstate_Closing = 5
)

// Desired state (ie, I want the door to open)
const (
	DesiredState_Closed = 0
	DesiredState_Open   = 1
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
// JSON->Go representation, and a "shadow" type to allow range()
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
	securityToken string
	applicationId string
	devices devices
	locations places
	inProgress bool
	debug bool
	machineReadable bool
}

// Helpers

func (f *MyQ) debugf(format string, a ...interface{}) (n int, err error) {
	if f.debug {
		return fmt.Fprintf(os.Stderr, format, a...)
	}
	return 0, nil
}

func (f *MyQ) doGet(rawurl string, v url.Values, s interface{}) (err error) {
	var r []byte
	var res *http.Response

	u, _ := url.Parse(rawurl)
	u.RawQuery = v.Encode()
	f.debugf("doGet():  URL -  %s\n", u.String())

	t := time.Now()
	if res, err = f.c.Get(u.String()); err != nil {
		f.debugf("Get() failed: %s\n", err)
		return err
	}
	d := time.Since(t)
	f.debugf("   HTTP Response: %s, in %s\n", res.Status, d)
	if res.StatusCode != http.StatusOK {
		return errors.New(res.Status)
	}
	r, err = ioutil.ReadAll(res.Body)
	res.Body.Close()

	if err != nil { 
		f.debugf("ReadAll() failed: %s\n", err)
		return err
	}
	return json.Unmarshal(r, &s)
}

func (f *MyQ) doPostRaw(rawurl string, v url.Values) (res *http.Response,
	err error) {

	if f.debug {
		u, _ := url.Parse(rawurl)
		u.RawQuery = v.Encode()
		f.debugf("doPost(): URL - %s\n", u)
	}
	t := time.Now()
	if res, err = f.c.PostForm(rawurl, v); err != nil {
		f.debugf("Post() failed: %s\n", err)
		return nil, err
	}
	d := time.Since(t)
	f.debugf("   HTTP Response: %s, in %s\n", res.Status, d)
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP Error: %s", res.Status)
	}

	return res, nil
}

func (f *MyQ) doPost(rawurl string, v url.Values, s interface{}) (err error) {
	var res *http.Response
	
	if res, err = f.doPostRaw(rawurl, v); err != nil {
		return err
	}
	r, err := ioutil.ReadAll(res.Body)
	res.Body.Close()

	return json.Unmarshal(r, &s)
}

func (d Device) String() string {
	d.Lastupdateddatetime = d.Lastupdateddatetime.Local()
	
	s := fmt.Sprintf("%s at %s (id %d) is %s since %s",
		d.Name, d.location, d.Myqdeviceid, d.Statename,
		d.Lastupdateddatetime.Format(time.UnixDate))

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
	return s
}

func (d Device) MachineString() string {
	d.Lastupdateddatetime = d.Lastupdateddatetime.Local()
	
	s := fmt.Sprintf("%s,%s,%d,%s,%d,%s,%s", d.Name, d.location,
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
	return s
}

func (f *MyQ) getAllGateways() (err error) {
	var j placeList

	// The oonly option is a current timestamp in ms
	v := url.Values{}
	v.Add("_", strconv.FormatInt(time.Now().UnixNano()/1000000, 10))
	if err = f. doGet(_BaseURL + "Gateway/GetAllGateways", v, &j);
	err != nil {
		return err
	}

	// Convert the JSON array to a map
	f.locations = make(places)
	for _, p := range(j.P) {
		f.locations[p.Gatewayid] = p
	}

	return err
}
	
func (f *MyQ) getAllDevices() (err error) {
	var d devices

	if err = f.getAllGateways(); err != nil {
		return err
	}

	v := url.Values{}
	v.Add("applicationId", f.applicationId)
	v.Add("securityToken", f.securityToken)
	v.Add("culture", _Culture)
	v.Add("brandName", _BrandName)
	if err = f.doGet(_BaseURL + "api/MyQDevices/GetAllDevices", v, &d);
	err != nil {
		return err
	}

	// Set the location for each device
	for i, x := range(d) {
		d[i].location = f.locations[x.Gatewayid].Name
	}

	f.devices = d
	return nil
}

func (f *MyQ) setDoorState(d Device, desiredstate int) (err error) {
	var t triggerStateChangeReturn

	f.debugf("SetDoorState: desiredstate = %d\n", desiredstate)
	if desiredstate != DesiredState_Open &&
	   desiredstate != DesiredState_Closed {
		return errors.New("Invalid door state")
	}

	v := url.Values{}
	v.Add("myQDeviceId", strconv.Itoa(d.Myqdeviceid))
	v.Add("attributename", "desireddoorstate")
	v.Add("attributevalue", strconv.Itoa(desiredstate))

	err = f.doPost(_BaseURL + "Device/TriggerStateChange", v, t)
	if t.Errormessage != "" {
		err = errors.New(t.Errormessage) 
		return err
	}

	return err
}

// Find the SecurityToken and ApplicationID in the HTML response from login POST
func findTokens(n *html.Node, securityToken *string, appID *string) (err error) {
	if n.DataAtom == atom.Input && n.Type == html.ElementNode {
		if n.Attr[0].Key == "type" && n.Attr[0].Val == "hidden" &&
		   n.Attr[1].Key == "id" && n.Attr[1].Val == "securityToken" {
		   *securityToken = n.Attr[2].Val
		}
	}

	if n.DataAtom == atom.Input && n.Type == html.ElementNode {
		if n.Attr[0].Key == "type" && n.Attr[0].Val == "hidden"&&
			n.Attr[1].Key == "id" && 
			n.Attr[1].Val == "ApplicationId" {
			*appID = n.Attr[2].Val
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		findTokens(c, securityToken, appID)
	}
	// TODO: Need to make sure that BOTH are set, otherwise return an error
	return err
}

func (f *MyQ) login(username string, password string) (err error) {
	var resp *http.Response
	var doc *html.Node

	err = nil
	v := url.Values{}
	v.Add("Email", username)
	v.Add("Password", password)
	
	if f.c.Jar, err = cookiejar.New(nil); err != nil {
		return fmt.Errorf("login(): Can't create CookieJar: %s", err)
	}
	f.c.Timeout = 60 * time.Second

	// This post needs to be done by hand, as the resulting HTML
	// needs to be parsed.
	if resp, err = f.doPostRaw(_BaseURL, v); err != nil {
		return fmt.Errorf("Post() failed: %s\n", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP Error: %s\n", resp.Status)
	}
	
	if doc, err = html.Parse(resp.Body); err != nil {
		return err
	}

	err = findTokens(doc, &f.securityToken, &f.applicationId)
	if f.securityToken == "" {
		return errors.New("Can't find securityToken in login response")
	}
	if f.applicationId == "" {
		return errors.New("Can't find applicationId in login response")
	}

	return err
}

func (p *place) string() string {
	return fmt.Sprintf("%s (ID %d)", p.Name, p.Gatewayid)
}

////////////////////////////////////////////////////////////////////////////
// PublicAPI
func (m *MyQ) New(username string, password string, debug bool,
	machineReadable bool) (err error) {
	m.debug = debug
	m.machineReadable = machineReadable 

	if err = m.login(username, password); err != nil {
		return err
	}
	
	if err = m.getAllDevices(); err != nil {
		return err
	}

	return err
}

func (m *MyQ) FindDoorByName(name string) (d Device, err error) {
	for _, d = range m.devices {
		if d.Name == name {
			return d, nil
		}
	}
	
	return d, fmt.Errorf("Device named '%s' not found", name)
}


func (m *MyQ) Update() error {
	return m.getAllDevices()
}

func (m *MyQ) ShowDoors() {
	for _, d := range m.devices {
		if m.machineReadable {
			fmt.Println(d.MachineString())
		} else {
			fmt.Println(d)
		}
	}
}

func (m *MyQ) ShowLocations() {
	for _, x := range m.locations {
		if m.machineReadable {
			fmt.Printf("%s,%d\n", x.Name, x.Gatewayid)
		} else {
			fmt.Println(x.string())
		}
	}
}

func (m *MyQ) DoorDetails(d Device) string {
	if m.machineReadable {
		return d.MachineString()
	} else {
		return d.String()
	}
}

func (m *MyQ) GetState(d Device) string {
	return d.Statename
}
	
func (m *MyQ) Open(d Device) error {
	state := m.GetState(d)
	if state == "Open" {
		return errors.New("Can't open, door is already open")
	}
	if state != "Closed" {
		return fmt.Errorf("Can't open, door is %s", state)
	}

	return m.setDoorState(d, DesiredState_Open)
}	

func (m *MyQ) Close(d Device) error {
	state := m.GetState(d)
	if state == "Closed" {
		return errors.New("Door already closed")
	}
	if state != "Open" {
		return fmt.Errorf("Can't close, door is %s", state)
	}

	return m.setDoorState(d, DesiredState_Closed)
}	

func (m *MyQ) ShowByState(s string) {
	for _, d := range m.devices {
		if d.Statename == s {
			if m.machineReadable {
				fmt.Println(d.MachineString())
			} else {
				fmt.Println(d)
			}
		}
	}
}
