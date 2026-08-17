package main

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/kitin/kitin/internal/notify"
)

func runNotifyCommand(args []string) {
	if len(args) == 0 {
		runNotifyStatus()
		return
	}
	switch args[0] {
	case "test":
		runNotifyTest()
	default:
		fmt.Println(amberStyle.Render("usage: kitin notify [test]"))
	}
}

func runNotifyStatus() {
	cfg := loadFullConfig()
	pushURL := strings.TrimSpace(cfg.Notifications.PushURL)
	if pushURL == "" {
		pushURL = strings.TrimSpace(os.Getenv("KITIN_PUSH_URL"))
	}

	fmt.Println()
	fmt.Println(boldSand.Render("  Phone push (local Wi‑Fi)"))
	fmt.Println()

	if ip := localLANIP(); ip != "" {
		fmt.Println(soilStyle.Render("  Your Mac on Wi‑Fi:"))
		fmt.Println(sandStyle.Render("    " + ip))
		fmt.Println()
		fmt.Println(soilStyle.Render("  Recommended push_url (ntfy on this Mac):"))
		fmt.Println(mossStyle.Render(fmt.Sprintf("    http://%s:2586/kitin-scans", ip)))
		fmt.Println()
	}

	if pushURL != "" {
		fmt.Println(soilStyle.Render("  Configured push_url:"))
		fmt.Println(sandStyle.Render("    " + pushURL))
	} else {
		fmt.Println(amberStyle.Render("  No push_url set yet."))
		fmt.Println(soilStyle.Render("  Run: kitin configure  (step 4)"))
		fmt.Println(soilStyle.Render("  Or edit ~/.kitin/config.json → notifications.push_url"))
	}

	fmt.Println()
	fmt.Println(soilStyle.Render("  Test: kitin notify test"))
	fmt.Println()
}

func runNotifyTest() {
	cfg := loadFullConfig()
	pushURL := strings.TrimSpace(cfg.Notifications.PushURL)
	if pushURL == "" {
		pushURL = strings.TrimSpace(os.Getenv("KITIN_PUSH_URL"))
	}
	if pushURL == "" {
		fmt.Println()
		fmt.Println(amberStyle.Render("  No push_url configured."))
		runNotifyStatus()
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println(soilStyle.Render("  Sending test push to:"))
	fmt.Println(sandStyle.Render("    " + pushURL))
	fmt.Println()

	if err := notify.SendTest(pushURL); err != nil {
		fmt.Println(rustStyle.Render("  ✗ failed: " + err.Error()))
		fmt.Println()
		fmt.Println(soilStyle.Render("  Check:"))
		fmt.Println(soilStyle.Render("    • phone and Mac on same Wi‑Fi"))
		fmt.Println(soilStyle.Render("    • ntfy server running on Mac (port 2586)"))
		fmt.Println(soilStyle.Render("    • ntfy app subscribed to the same topic"))
		fmt.Println()
		os.Exit(1)
	}

	fmt.Println(okStyle.Render("  ✔ test push sent — check your phone"))
	fmt.Println()
}

func localLANIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue
			}
			s := ip.String()
			if strings.HasPrefix(s, "192.168.") || strings.HasPrefix(s, "10.") || strings.HasPrefix(s, "172.") {
				return s
			}
		}
	}
	return ""
}
