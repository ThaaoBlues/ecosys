package networking

import (
	"os"
	"qsync/bdd"
	"strings"
	"time"

	"github.com/hashicorp/mdns"
)

type ZeroConfService struct {
	Server *mdns.Server
}

func (zcs *ZeroConfService) Browse() {

	var acces bdd.AccesBdd
	acces.InitConnection()
	defer acces.CloseConnection()

	// check if we want to just browse the network or actually update a table
	old_connected_devices := acces.GetOnlineDevices()
	new_connected_devices := GetNetworkDevices()

	// first, we put all linked devices state to false
	for _, old_device := range old_connected_devices {
		acces.SetDeviceConnectionState(old_device, false)
	}
	// then, we put all linked and connected devices state to true
	for _, new_device := range new_connected_devices {
		if acces.IsDeviceLinked(new_device["device_id"]) {
			acces.SetDeviceConnectionState(new_device["device_id"], true, new_device["ip_addr"])
		}
	}

	// so now, we have a fully up to date device table without making double loops or shits like that :D
}

func (zcs *ZeroConfService) UpdateDevicesConnectionStateLoop() {
	for {
		zcs.Browse()
		time.Sleep(10 * time.Second)
	}
}

func (zcs *ZeroConfService) Register() {

	//var err error

	var acces bdd.AccesBdd
	acces.InitConnection()

	// Setup our service export
	host, _ := os.Hostname()
	info := []string{
		"version=0.0.1-PreAlpha",
		"device_id=" + acces.GetMyDeviceId(),
	}
	service, _ := mdns.NewMDNSService(host, "_qsync._tcp.local", "", "", 8274, nil, info)

	// Create the mDNS server, defer shutdown
	server, _ := mdns.NewServer(&mdns.Config{Zone: service})

	zcs.Server = server
}

func (zcs *ZeroConfService) Shutdown() {
	zcs.Server.Shutdown()
}

func GetNetworkDevices() []map[string]string {

	devices_list := make([]map[string]string, 0)

	// Make a channel for results and start listening
	entriesCh := make(chan *mdns.ServiceEntry, 4)
	go func() {
		for entry := range entriesCh {
			dev := make(map[string]string, 0)
			dev["host"] = entry.Host + "local"
			dev["ip_addr"] = entry.AddrV4.String()
			dev["version"] = strings.Split(entry.InfoFields[0], "=")[1]
			dev["device_id"] = strings.Split(entry.InfoFields[1], "=")[1]
			devices_list = append(devices_list, dev)

		}
	}()

	// Start the lookup
	//log.SetOutput(io.Discard)
	mdns.Lookup("_qsync._tcp.local.", entriesCh)
	close(entriesCh)

	return devices_list
}
