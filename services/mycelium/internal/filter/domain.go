package filter

import (
	"net/url"
	"strings"
)

type DomainFilter struct {
	domains map[string]bool
}

func NewDomainFilter(domains []string) *DomainFilter {
	domainsMap := map[string]bool{}
	for _, d := range domains {
		domainsMap[strings.ToLower(d)] = true
	}
	return &DomainFilter{domains: domainsMap}
}

func (f *DomainFilter) Filter(u *url.URL) bool {
	if u == nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return false
	}

	// direct match
	if _, found := f.domains[host]; found {
		return true
	}

	// check parent domains (e.g., sub.example.com -> example.com)
	parts := strings.Split(host, ".")
	for i := 1; i < len(parts)-1; i++ {
		parent := strings.Join(parts[i:], ".")
		if _, found := f.domains[parent]; found {
			return true
		}
	}

	return false
}
