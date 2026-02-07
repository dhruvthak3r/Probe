package monitor

import (
	"net/http"
	"net/textproto"
	"strings"
)

func ValidateResponseStatusCode(respStatusCode int, acceptedStatusCode []int) bool {
	for _, c := range acceptedStatusCode {
		if c == respStatusCode {
			return true
		}
	}

	return false
}

func ValidateResponseHeaders(expected map[string][]string, respHeaders http.Header) bool {
	for expectedKey, expectedValues := range expected {

		canonicalKey := textproto.CanonicalMIMEHeaderKey(expectedKey)
		actualValues := respHeaders[canonicalKey]

		if len(actualValues) == 0 {
			return false
		}

		for _, expectedVal := range expectedValues {
			expectedVal = strings.TrimSpace(expectedVal)
			matchFound := false

			for _, actualVal := range actualValues {
				actualVal = strings.TrimSpace(actualVal)

				if actualVal == expectedVal || strings.Contains(actualVal, expectedVal) {
					matchFound = true
					break
				}
			}

			if !matchFound {
				return false
			}
		}
	}

	return true
}
