package networking

import (
	"context"
	"log"
	"os"
	"os/signal"
	"qsync/bdd"
	"strings"
	"syscall"
	"time"

	"github.com/grandcat/zeroconf"
)

type ZeroConfService struct {
	Resolver *zeroconf.Resolver
	Server   *zeroconf.Server
}

func (zcs *ZeroConfService) Browse() {
	// Discover all services on the network (e.g. _workstation._tcp)
	var err error
	zcs.Resolver, err = zeroconf.NewResolver(nil)

	if err != nil {
		log.Fatalln("Failed to initialize resolver:", err.Error())
	}

	entries := make(chan *zeroconf.ServiceEntry)

	var acces bdd.AccesBdd
	acces.InitConnection()
	defer acces.CloseConnection()

	old_connected_devices := acces.GetOnlineDevices()
	var new_connected_devices []string

	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			log.Println("Found QSync service running on the network :")
			log.Println("\tIP : ", entry.AddrIPv4)

			device_metadata := map[string]string{
				"version":  strings.Split(entry.Text[0], "=")[1],
				"id":       strings.Split(entry.Text[1], "=")[1],
				"hostname": entry.HostName[0 : len(entry.HostName)-1],
			}

			log.Println("\tVERSION : ", device_metadata["version"])
			log.Println("\tID : ", device_metadata["id"])
			log.Println("\tHOSTNAME : ", device_metadata["hostname"])

			if acces.IsDeviceLinked(device_metadata["id"]) {
				new_connected_devices = append(new_connected_devices, device_metadata["id"])
				acces.SetDeviceConnectionState(device_metadata["id"], true)
				acces.SetDeviceIP(device_metadata["id"], string(entry.AddrIPv4[0]))
			} else {

				if device_metadata["id"] == acces.GetMyDeviceId() {
					log.Println("Hey !! That's your device right here !!")
				} else {
					log.Println("This device is not linked.")
				}
			}

		}
		log.Println("No more entries.")
	}(entries)

	var found bool = false
	for _, device := range old_connected_devices {
		for _, new_device := range new_connected_devices {
			if device == new_device {
				found = true
				break
			}
		}

		if !found {
			log.Println("Device disconneted : ", device)
			acces.SetDeviceConnectionState(device, false)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	err = zcs.Resolver.Browse(ctx, "_qsync._tcp", "local.", entries)
	if err != nil {
		log.Fatalln("Failed to browse:", err.Error())
	}

	<-ctx.Done()

}

func (zcs *ZeroConfService) Register() {

	var err error

	var acces bdd.AccesBdd
	acces.InitConnection()
	defer acces.CloseConnection()

	zcs.Server, err = zeroconf.Register("QSync", "_qsync._tcp", "local.", 8274, []string{
		"version=0.0.1-PreAlpha",
		"device_id=" + acces.GetMyDeviceId(),
	}, nil)

	if err != nil {
		panic(err)
	}
	defer zcs.Server.Shutdown()

	// Clean exit.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	select {
	case <-sig:
		// Exit by user
		zcs.Server.Shutdown()
	case <-time.After(time.Second * 120):
		// Exit by timeout
		zcs.Server.Shutdown()
	}

	log.Println("Shutting down.")
}

func GetNetworkDevices() []map[string]string {
	var zcs ZeroConfService

	devices_list := make([]map[string]string, 0)

	// register this device
	go zcs.Register()

	// Discover all services on the network (e.g. _workstation._tcp)
	var err error
	zcs.Resolver, err = zeroconf.NewResolver(nil)

	if err != nil {
		log.Fatalln("Failed to initialize resolver:", err.Error())
	}

	entries := make(chan *zeroconf.ServiceEntry)

	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {

			device_metadata := map[string]string{
				"version":  strings.Split(entry.Text[0], "=")[1],
				"id":       strings.Split(entry.Text[1], "=")[1],
				"hostname": entry.HostName[0 : len(entry.HostName)-1],
			}

			devices_list = append(devices_list, device_metadata)

		}
		log.Println("No more entries.")
	}(entries)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	err = zcs.Resolver.Browse(ctx, "_qsync._tcp", "local.", entries)
	if err != nil {
		log.Fatalln("Failed to browse:", err.Error())
	}

	<-ctx.Done()

	return devices_list
}
