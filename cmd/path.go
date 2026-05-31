package cmd

import "net/url"

func pathSegment(value string) string {
	return url.PathEscape(value)
}
