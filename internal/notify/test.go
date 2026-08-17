package notify

import "time"

// SendTest sends a minimal report to verify push_url / phone setup.
func SendTest(url string) error {
	return SendPush(url, ScanReport{
		ScanPath:  "kitin notify test",
		FileCount: 1,
		Exposure:  "$0",
		High:      0,
		Med:       0,
		Low:       1,
		ScannedAt: time.Now(),
		Findings: []Finding{{
			Severity: "low",
			Message:  "kitin push test — if you see this on your phone, it works!",
			Fix:      "run kitin scan to get real reports",
		}},
	})
}
