package cmd

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// timeNow is the clock for "since" times; tests replace it.
var timeNow = time.Now

// statusWord prints a Hub status as the console shows it: "online" is
// "Online" and "stale" is "Not responding". JSON keeps the Hub's value.
func statusWord(value any) string {
	text := strings.TrimSpace(cellText(value))
	switch strings.ToLower(text) {
	case "stale", "unresponsive":
		return "Not responding"
	}
	text = strings.ReplaceAll(text, "_", " ")
	first, size := utf8.DecodeRuneInString(text)
	if first == utf8.RuneError {
		return text
	}
	return string(unicode.ToUpper(first)) + text[size:]
}

// deviceStatus adds when a device was last seen to Offline and Not
// responding, as the console does: "Offline since 21:12".
func deviceStatus(row map[string]any) string {
	word := statusWord(lookupPath(row, "status"))
	switch word {
	case "Offline", "Not responding":
		if since := sinceText(lookupPath(row, "last_seen_at")); since != "" {
			return word + " since " + since
		}
	}
	return word
}

// sinceText prints a past RFC 3339 time the way the console's status words
// do: the time of day within the last 24 hours, otherwise the date. A missing
// or zero time, which the Hub sends for a device never seen, prints nothing.
func sinceText(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	at, err := time.Parse(time.RFC3339Nano, text)
	if err != nil || at.Unix() <= 0 {
		return ""
	}
	now := timeNow()
	local := at.Local()
	switch {
	case now.Sub(at) < 24*time.Hour:
		return local.Format("15:04")
	case local.Year() == now.Local().Year():
		return local.Format("2 Jan")
	default:
		return local.Format("2 Jan 2006")
	}
}
