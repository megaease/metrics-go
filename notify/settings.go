package notify

const (
	DefaultIconURL    = "https://megaease.com/favicon.png"
	DefaultTimeFormat = "2006-01-02 15:04:05 Z0700"
)

// Status is the status of Probe
type Status int

// The status of a probe
const (
	StatusUnknown Status = iota
	StatusSuccess
	StatusFailure
)

var (
	toEmoji = map[Status]string{
		StatusUnknown: "⛔️",
		StatusSuccess: "✅",
		StatusFailure: "❌",
	}
)

// Emoji convert the status to emoji
func (s *Status) Emoji() string {
	if val, ok := toEmoji[*s]; ok {
		return val
	}
	return "⛔️"
}
