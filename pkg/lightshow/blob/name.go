package blob

import "strconv"

// ResolveName expands a resource name containing a "$" expression, e.g. "Fly_$C+1".
// After the '$': 'F' = frame, 'C' = frame - start, then any number of "<op><number>" terms
// (+ - * / %), applied left to right. Invalid expressions resolve to "".
func ResolveName(name string, frame, start int) string {
	dollar := -1
	for i := 0; i < len(name); i++ {
		if name[i] == '$' {
			dollar = i
			break
		}
	}
	if dollar < 0 {
		return name
	}
	const (
		wantVar = iota
		wantOp
		wantDigit
		inNumber
	)
	state, value, number := wantVar, 0, 0
	var op byte
	for i := dollar + 1; i < len(name); i++ {
		ch := name[i]
		switch state {
		case wantVar:
			switch ch {
			case 'F':
				value = frame
			case 'C':
				value = frame - start
			default:
				return ""
			}
			state = wantOp
		case wantOp:
			if !isOp(ch) {
				return ""
			}
			op, state = ch, wantDigit
		case wantDigit:
			if isDigit(ch) {
				number, state = int(ch-'0'), inNumber
			}
		case inNumber:
			if isDigit(ch) {
				number = number*10 + int(ch-'0')
				continue
			}
			if !isOp(ch) {
				return ""
			}
			var ok bool
			if value, ok = apply(value, op, number); !ok {
				return ""
			}
			op, state = ch, wantDigit
		}
	}
	switch state {
	case wantOp:
	case inNumber:
		var ok bool
		if value, ok = apply(value, op, number); !ok {
			return ""
		}
	default:
		return ""
	}
	return name[:dollar] + strconv.Itoa(value)
}

func isDigit(ch byte) bool { return ch >= '0' && ch <= '9' }

func isOp(ch byte) bool {
	return ch == '+' || ch == '-' || ch == '*' || ch == '/' || ch == '%'
}

func apply(value int, op byte, n int) (int, bool) {
	switch op {
	case '+':
		return value + n, true
	case '-':
		return value - n, true
	case '*':
		return value * n, true
	case '/':
		if n == 0 {
			return 0, false
		}
		return value / n, true
	case '%':
		if n == 0 {
			return 0, false
		}
		return value % n, true
	}
	return 0, false
}
