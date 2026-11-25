package monitor

import (
	"cmp"
	"fmt"
	"io"
	"strings"
)

func PrintWiFi(out io.Writer) {
	adapters, err := readWiFi()
	if err != nil {
		PrintError(out, "read Wi-Fi information", err)
		return
	}
	if len(adapters) == 0 {
		fmt.Fprintln(out, "No Wi-Fi adapters found")
		return
	}

	for i, info := range adapters {
		if i > 0 {
			fmt.Fprintln(out)
		}
		fmt.Fprintln(out, info.model)
		if !info.connected {
			fmt.Fprintln(out, "Status: disconnected")
			continue
		}

		if info.signalAvailable {
			signal := float64(min(info.signalQuality, 100))
			bar, _ := UsageBar(signal)
			fmt.Fprintf(out, "Signal strength: [%s] %.0f%%\n", bar, signal)
		} else if info.linkUsageAvailable {
			bar, _ := UsageBar(info.linkUsagePercent)
			fmt.Fprintf(out, "Throughput: [%s] %.1f%% of link speed\n", bar, info.linkUsagePercent)
		} else {
			fmt.Fprintln(out, "Signal strength: unavailable")
		}

		send, receive := unavailable, unavailable
		if info.trafficAvailable {
			send = formatWiFiRate(info.sendBitsPerSecond)
			receive = formatWiFiRate(info.receiveBitsPerSecond)
		}
		fmt.Fprintf(out, "Send: %s | Receive: %s\n", send, receive)

		switch {
		case info.ssid != "":
			fmt.Fprintf(out, "SSID: %q\n", info.ssid)
		case info.locationAccessDenied:
			fmt.Fprintln(out, "SSID: unavailable (Windows location permission required)")
		default:
			fmt.Fprintln(out, "SSID: unavailable")
		}
		fmt.Fprintf(out, "Connection type: %s\n", cmp.Or(info.connectionType, unavailable))
		fmt.Fprintf(out, "IPv4: %s\n", cmp.Or(strings.Join(info.ipv4, ", "), unavailable))
		fmt.Fprintf(out, "IPv6: %s\n", cmp.Or(strings.Join(info.ipv6, ", "), unavailable))
	}
}

func formatWiFiRate(bitsPerSecond float64) string {
	if bitsPerSecond >= 1_000_000 {
		return fmt.Sprintf("%.1f Mbps", bitsPerSecond/1_000_000)
	}
	return fmt.Sprintf("%.1f Kbps", bitsPerSecond/1000)
}
