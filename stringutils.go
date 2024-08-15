package main

import "strings"

const q = `"`

// capitalize the first character of the string and make all other characters lowercase
func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

// betweenQuotes can return what's between the double quotes in the given string.
// If fewer than two double quotes are found, return the original string.
func betweenQuotes(orig string) string {
	if strings.Count(orig, q) >= 2 {
		posa := strings.Index(orig, q) + 1
		posb := strings.LastIndex(orig, q)
		if posa >= posb {
			return orig
		}
		return orig[posa:posb]
	}
	return orig
}
