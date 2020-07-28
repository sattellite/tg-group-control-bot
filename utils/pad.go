package utils

// PadRight adding pad value at right
func PadRight(str, pad string, length int) string {
	for {
		if len(str) >= length {
			return str[0:length]
		}
		str += pad
	}
}

// PadLeft adding pad value at left
func PadLeft(str, pad string, length int) string {
	for {
		if len(str) >= length {
			return str[0:length]
		}
		str = pad + str
	}
}
