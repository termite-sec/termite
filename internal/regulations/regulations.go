package regulations

import "strings"

type Regulation struct {
	Key      string
	Label    string
	Desc     string
	Fine     string
	Category []string
}

var All = []Regulation{
	{"hipaa", "HIPAA", "Health Insurance Portability and Accountability Act", "fines up to $1.9M per year per violation category", []string{"federal", "health"}},
	{"coppa", "COPPA", "Children's Online Privacy Protection Act", "fines up to $51,744 per violation", []string{"federal", "children", "data"}},
	{"can-spam", "CAN-SPAM", "Controlling the Assault of Non-Solicited Pornography and Marketing Act", "fines up to $51,744 per email", []string{"federal", "data"}},
	{"tcpa", "TCPA", "Telephone Consumer Protection Act", "fines $500–$1,500 per violation", []string{"federal", "data"}},
	{"vppa", "VPPA", "Video Privacy Protection Act", "fines up to $2,500 per violation", []string{"federal", "data"}},
	{"glba", "GLBA", "Gramm-Leach-Bliley Act — financial data privacy", "fines up to $100,000 per violation", []string{"federal", "financial"}},
	{"eo14117", "Executive Order 14117", "Preventing Access to Americans' Bulk Sensitive Personal Data", "civil and criminal penalties", []string{"federal", "data"}},
	{"nist-800-53", "NIST 800-53", "US federal security and privacy controls framework", "federal contract loss, audit findings", []string{"federal"}},
	{"gdpr", "GDPR", "General Data Protection Regulation (EU)", "fines up to €20M or 4% of global annual revenue", []string{"international", "data"}},
	{"ccpa", "CCPA", "California Consumer Privacy Act", "fines up to $7,500 per intentional violation", []string{"state", "data", "california"}},
	{"cpra", "CPRA", "California Privacy Rights Act (extends CCPA)", "fines up to $7,500 per violation, triples for minors", []string{"state", "data", "california"}},
	{"caadc", "CAADC", "California Age-Appropriate Design Code (SB 362)", "fines up to $7,500 per affected child per violation", []string{"state", "children", "california"}},
	{"cipa", "CIPA", "California Invasion of Privacy Act", "fines up to $5,000 per violation", []string{"state", "data", "california"}},
	{"shine-the-light", "Shine the Light", "California data sharing disclosure law", "fines up to $3,000 per violation", []string{"state", "data", "california"}},
	{"caloppa", "CalOPPA", "California Online Privacy Protection Act — privacy policy requirement", "fines up to $2,500 per violation", []string{"state", "data", "california"}},
	{"ca-delete-act", "California Delete Act", "SB 362 — data broker deletion requests", "fines up to $200/day per consumer per data broker", []string{"state", "data", "california", "broker"}},
	{"ca-iot-sb327", "California IoT Security Law (SB 327)", "Security requirements for connected devices", "injunctive relief, civil penalties", []string{"state", "california"}},
	{"mhmd", "MHMD", "Washington My Health My Data Act", "fines up to $7,500 per violation, private right of action", []string{"state", "health"}},
	{"nv-health", "Nevada Consumer Health Data Privacy Law", "Nevada health data privacy protections", "civil penalties up to $15,000 per violation", []string{"state", "health"}},
	{"bipa", "BIPA", "Illinois Biometric Information Privacy Act", "fines $1,000–$5,000 per violation, private right of action", []string{"state", "biometric"}},
	{"ny-shield", "New York SHIELD Act", "Stop Hacks and Improve Electronic Data Security Act", "fines up to $250,000", []string{"state", "data"}},
	{"nydfs", "NYDFS", "NY Dept. of Financial Services Cybersecurity Regulations — 23 NYCRR 500", "fines up to $1,000 per violation per day", []string{"state", "financial"}},
	{"pci-dss", "PCI-DSS", "Payment Card Industry Data Security Standard", "fines $5,000–$100,000 per month", []string{"financial"}},
	{"soc2", "SOC2", "Service Organization Control 2 — enterprise compliance", "loss of enterprise contracts, audit failures", []string{"financial"}},
	{"oh-sb200", "Ohio SB 200", "Ohio Cybersecurity Safe Harbor — affirmative defense if compliant", "safe harbor from tort liability", []string{"state", "financial"}},
	{"co-privacy", "Colorado Privacy Act", "Consumer data rights and controller obligations", "fines up to $20,000 per violation", []string{"state", "data"}},
	{"ct-privacy", "Connecticut Data Privacy Act", "Consumer data rights for CT residents", "fines up to $5,000 per violation", []string{"state", "data"}},
	{"de-privacy", "Delaware Personal Data Privacy Act", "Consumer data privacy rights for DE residents", "fines up to $10,000 per violation", []string{"state", "data"}},
	{"fl-privacy", "Florida Data Privacy and Security Act", "SB 262 — consumer data rights for FL residents", "fines up to $50,000 per violation", []string{"state", "data"}},
	{"in-privacy", "Indiana Consumer Data Protection Act", "Consumer data privacy rights for IN residents", "fines up to $7,500 per violation", []string{"state", "data"}},
	{"ia-privacy", "Iowa Consumer Data Protection Act", "Consumer data privacy rights for IA residents", "fines up to $7,500 per violation", []string{"state", "data"}},
	{"ky-privacy", "Kentucky Consumer Data Protection Act", "Consumer data privacy rights for KY residents", "fines up to $7,500 per violation", []string{"state", "data"}},
	{"md-privacy", "Maryland Online Data Privacy Act", "Consumer data privacy rights for MD residents", "fines up to $10,000 per violation", []string{"state", "data"}},
	{"mn-privacy", "Minnesota Consumer Data Privacy Act", "Consumer data privacy rights for MN residents", "fines up to $7,500 per violation", []string{"state", "data"}},
	{"mt-privacy", "Montana Consumer Data Privacy Act", "Consumer data privacy rights for MT residents", "fines up to $7,500 per violation", []string{"state", "data"}},
	{"ne-privacy", "Nebraska Data Privacy Act", "Consumer data privacy rights for NE residents", "fines up to $7,500 per violation", []string{"state", "data"}},
	{"nh-privacy", "New Hampshire Consumer Expectation of Privacy Act", "Consumer data privacy for NH residents", "fines up to $10,000 per violation", []string{"state", "data"}},
	{"nj-privacy", "New Jersey Personal Data Privacy Act", "Consumer data privacy rights for NJ residents", "fines up to $10,000 per violation", []string{"state", "data"}},
	{"or-privacy", "Oregon Consumer Privacy Act", "Consumer data privacy rights for OR residents", "fines up to $7,500 per violation", []string{"state", "data"}},
	{"ri-privacy", "Rhode Island Data Transparency and Privacy Protection Act", "Consumer data privacy for RI residents", "fines up to $10,000 per violation", []string{"state", "data"}},
	{"tn-privacy", "Tennessee Information Protection Act", "Consumer data privacy rights for TN residents", "fines up to $15,000 per violation", []string{"state", "data"}},
	{"tx-privacy", "Texas Data Privacy and Security Act", "Consumer data privacy rights for TX residents", "fines up to $7,500 per violation", []string{"state", "data"}},
	{"ut-privacy", "Utah Consumer Privacy Act", "Consumer data privacy rights for UT residents", "fines up to $7,500 per violation", []string{"state", "data"}},
	{"va-privacy", "Virginia Consumer Data Protection Act", "Consumer data privacy rights for VA residents", "fines up to $7,500 per violation", []string{"state", "data"}},
	{"vt-broker", "Vermont Data Broker Registration Law", "Data brokers must register and meet security standards", "fines up to $10,000 per day", []string{"state", "data", "broker"}},
	{"or-broker", "Oregon Data Broker Law", "Data broker registration and consumer opt-out", "fines up to $1,000 per day", []string{"state", "data", "broker"}},
	{"tx-broker", "Texas Data Broker Law", "Data broker registration requirements in Texas", "fines up to $10,000 per violation", []string{"state", "data", "broker"}},
}

func ByKey() map[string]Regulation {
	m := make(map[string]Regulation, len(All))
	for _, r := range All {
		m[r.Key] = r
	}
	return m
}

func FormatForPrompt(modes []string) string {
	regMap := ByKey()
	parts := make([]string, 0, len(modes))
	for _, mode := range modes {
		if r, ok := regMap[mode]; ok {
			parts = append(parts, r.Label+" — "+r.Desc+" (exposure: "+r.Fine+")")
			continue
		}
		parts = append(parts, mode)
	}
	return strings.Join(parts, "; ")
}
