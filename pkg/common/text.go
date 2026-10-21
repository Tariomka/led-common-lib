package common

// Flag formats a boolean as "1" or "0", like the factory firmware's log lines.
func Flag(value bool) string {
	if value {
		return "1"
	}

	return "0"
}
