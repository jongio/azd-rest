package service

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

// lowQuotaThreshold is the remaining-request count at or below which
// --show-throttle prints a low-quota warning.
const lowQuotaThreshold = 100

// throttleHeaders lists the Azure rate-limit and quota response headers that
// --show-throttle surfaces. Header lookup is case-insensitive. Keeping the
// list here makes it easy to add new headers as Azure introduces them.
var throttleHeaders = []string{
	"x-ms-ratelimit-remaining-subscription-reads",
	"x-ms-ratelimit-remaining-subscription-writes",
	"x-ms-ratelimit-remaining-subscription-deletes",
	"x-ms-ratelimit-remaining-subscription-resource-requests",
	"x-ms-ratelimit-remaining-subscription-resource-entities-read",
	"x-ms-ratelimit-remaining-tenant-reads",
	"x-ms-ratelimit-remaining-tenant-writes",
	"x-ms-ratelimit-remaining-resource",
	"x-ms-user-quota-remaining",
}

// writeThrottleInfo scans response headers for the known Azure rate-limit and
// quota headers and prints a summary to w. Nothing is printed when none are
// present. Any value that parses as an integer at or below lowQuotaThreshold
// also produces a warning line.
func writeThrottleInfo(w io.Writer, headers http.Header) {
	type entry struct {
		name  string
		value string
	}

	var found []entry
	for _, name := range throttleHeaders {
		if value := headers.Get(name); value != "" {
			found = append(found, entry{name: name, value: value})
		}
	}
	if len(found) == 0 {
		return
	}

	sort.Slice(found, func(i, j int) bool { return found[i].name < found[j].name })

	fmt.Fprintln(w, "\nRate limit / quota:")
	var low []entry
	for _, e := range found {
		fmt.Fprintf(w, "  %s: %s\n", e.name, e.value)
		if n, err := strconv.Atoi(strings.TrimSpace(e.value)); err == nil && n <= lowQuotaThreshold {
			low = append(low, e)
		}
	}
	for _, e := range low {
		fmt.Fprintf(w, "Warning: low remaining quota for %s (%s).\n", e.name, e.value)
	}
}
