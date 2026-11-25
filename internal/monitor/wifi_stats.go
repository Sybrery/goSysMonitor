package monitor

import (
	"net"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type wifiInformation struct {
	model                string
	connected            bool
	ssid                 string
	connectionType       string
	ipv4                 []string
	ipv6                 []string
	signalQuality        uint32
	signalAvailable      bool
	sendBitsPerSecond    float64
	receiveBitsPerSecond float64
	trafficAvailable     bool
	linkUsagePercent     float64
	linkUsageAvailable   bool
	locationAccessDenied bool
}

func readWiFiAdapters() ([]windows.MibIfRow2, error) {
	var table *windows.MibIfTable2
	if err := windows.GetIfTable2Ex(windows.MibIfTableNormal, &table); err != nil {
		return nil, err
	}
	defer windows.FreeMibTable(unsafe.Pointer(table))
	const hardwareInterface = 1
	var adapters []windows.MibIfRow2
	for _, row := range unsafe.Slice(&table.Table[0], table.NumEntries) {
		if row.Type != windows.IF_TYPE_IEEE80211 || row.InterfaceAndOperStatusFlags&hardwareInterface == 0 {
			continue
		}
		adapters = append(adapters, row)
	}
	return adapters, nil
}

func readWiFi() ([]wifiInformation, error) {
	adapters, err := readWiFiAdapters()
	if err != nil {
		return nil, err
	}
	started := time.Now()
	const unknownLinkSpeed = ^uint64(0)
	var hasConnection bool
	for _, row := range adapters {
		hasConnection = hasConnection || row.OperStatus == windows.IfOperStatusUp
	}

	wlan := windows.NewLazySystemDLL("wlanapi.dll")
	var client windows.Handle
	if hasConnection {
		time.Sleep(time.Second)
		client = openWiFiClient(wlan)
		if client != 0 {
			defer wlan.NewProc("WlanCloseHandle").Call(uintptr(client), 0)
		}
	}

	results := make([]wifiInformation, 0, len(adapters))
	for _, row := range adapters {
		info := wifiInformation{
			model:     windows.UTF16ToString(row.Description[:]),
			connected: row.OperStatus == windows.IfOperStatusUp,
		}
		if info.connected {
			current := windows.MibIfRow2{InterfaceLuid: row.InterfaceLuid}
			if err := windows.GetIfEntry2Ex(windows.MibIfEntryNormal, &current); err == nil {
				info.connected = current.OperStatus == windows.IfOperStatusUp
				if info.connected && current.OutOctets >= row.OutOctets && current.InOctets >= row.InOctets {
					elapsed := time.Since(started).Seconds()
					info.sendBitsPerSecond = float64(current.OutOctets-row.OutOctets) * 8 / elapsed
					info.receiveBitsPerSecond = float64(current.InOctets-row.InOctets) * 8 / elapsed
					info.trafficAvailable = true
					tx, rx := current.TransmitLinkSpeed, current.ReceiveLinkSpeed
					if tx > 0 && rx > 0 && tx != unknownLinkSpeed && rx != unknownLinkSpeed {
						sendShare := info.sendBitsPerSecond / float64(tx)
						receiveShare := info.receiveBitsPerSecond / float64(rx)
						info.linkUsagePercent = min(100, max(sendShare, receiveShare)*100)
						info.linkUsageAvailable = true
					}
				}
			}
		}
		if info.connected {
			for _, ip := range wifiInterfaceIPs(row.InterfaceIndex) {
				if ip.To4() != nil {
					info.ipv4 = append(info.ipv4, ip.String())
				} else {
					info.ipv6 = append(info.ipv6, ip.String())
				}
			}
			if client != 0 {
				readWiFiConnection(wlan, client, &row.InterfaceGuid, &info)
			}
		}
		results = append(results, info)
	}
	return results, nil
}

func wifiInterfaceIPs(index uint32) []net.IP {
	nic := net.Interface{Index: int(index)}
	addresses, _ := nic.Addrs()
	var ips []net.IP
	for _, address := range addresses {
		if ip, ok := address.(*net.IPNet); ok {
			ips = append(ips, ip.IP)
		}
	}
	return ips
}

func openWiFiClient(wlan *windows.LazyDLL) windows.Handle {
	open := wlan.NewProc("WlanOpenHandle")
	if err := open.Find(); err != nil {
		return 0
	}
	var version uint32
	var client windows.Handle
	status, _, _ := open.Call(2, 0, uintptr(unsafe.Pointer(&version)), uintptr(unsafe.Pointer(&client)))
	if uint32(status) != 0 {
		return 0
	}
	return client
}

type wlanConnectionAttributes struct {
	state      uint32
	_          uint32
	_          [256]uint16
	ssidLength uint32
	ssid       [32]byte
	_          uint32
	_          [6]byte
	_          [2]byte
	phyType    uint32
	_          uint32
	signal     uint32
}

func readWiFiConnection(wlan *windows.LazyDLL, client windows.Handle, guid *windows.GUID, info *wifiInformation) {
	const (
		currentConnection = 7
		realtimeQuality   = 19
		wlanConnected     = 1
	)
	query := wlan.NewProc("WlanQueryInterface")
	free := wlan.NewProc("WlanFreeMemory")
	data, size, status := queryWiFi(query, client, guid, currentConnection)
	if status == 0 && data != nil {
		defer free.Call(uintptr(data))
		if uintptr(size) >= unsafe.Sizeof(wlanConnectionAttributes{}) {
			current := (*wlanConnectionAttributes)(data)
			if current.state == wlanConnected {
				if current.ssidLength <= uint32(len(current.ssid)) {
					info.ssid = string(current.ssid[:current.ssidLength])
				}
				info.connectionType = wifiConnectionType(current.phyType)
				info.signalQuality = current.signal
				info.signalAvailable = true
				return
			}
		}
	}
	info.locationAccessDenied = status == uint32(windows.ERROR_ACCESS_DENIED)

	data, size, status = queryWiFi(query, client, guid, realtimeQuality)
	if status == 0 && data != nil {
		defer free.Call(uintptr(data))
		if size >= 8 {
			quality := (*[2]uint32)(data)
			info.connectionType = wifiConnectionType(quality[0])
			info.signalQuality = quality[1]
			info.signalAvailable = true
		}
	}
}

func queryWiFi(query *windows.LazyProc, client windows.Handle, guid *windows.GUID, opcode uintptr) (unsafe.Pointer, uint32, uint32) {
	var size uint32
	var data unsafe.Pointer
	status, _, _ := query.Call(uintptr(client), uintptr(unsafe.Pointer(guid)), opcode, 0,
		uintptr(unsafe.Pointer(&size)), uintptr(unsafe.Pointer(&data)), 0)
	return data, size, uint32(status)
}

func wifiConnectionType(phyType uint32) string {
	return map[uint32]string{
		1:  "802.11 (FHSS)",
		2:  "802.11 (DSSS)",
		3:  "802.11 (IR)",
		4:  "802.11a",
		5:  "802.11b",
		6:  "802.11g",
		7:  "802.11n",
		8:  "802.11ac",
		9:  "802.11ad",
		10: "802.11ax",
		11: "802.11be",
	}[phyType]
}
