package utils

import (
	"math/rand"
	"strings"
)

func GenID(length int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz-_"

	id := make([]byte, length)

	for i := range id {
		id[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(id)
}

func MinifyTemplate(templ_text string) string {

	minified_text := templ_text

	minified_text = strings.ReplaceAll(minified_text, "\n", "")

	minified_text = strings.ReplaceAll(minified_text, ";  ", ";")

	return minified_text
}
