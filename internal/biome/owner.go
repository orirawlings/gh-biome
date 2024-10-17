package biome

import (
	"fmt"
	"path"
	"strings"
)

const (
	defaultOwnerHost = "github.com"
)

// Owner of GitHub repositories, i.e. a GitHub user or organization.
type Owner struct {

	// host is the GitHub server name, ex. `github.com`. If empty, `github.com`
	// is assumed.
	host string

	// name of the GitHub user or organziation.
	name string
}

// ParseOwner identifies the GitHub user or organization given a reference,
// typically typed in as a command line argument.
//
// GitHub owners are specified with the following format, where <host> is the
// GitHub server name and <owner-name> is the name of the GitHub user or
// organziation. If <host> is omitted, "github.com" is assumed.
//
//	[https://][<host>/]<name>
//
// Examples:
//
//	orirawlings
//	github.com/orirawlings
//	https://github.com/orirawlings
func ParseOwner(ownerRef string) (Owner, error) {
	var protocolIncluded bool
	s, protocolIncluded := strings.CutPrefix(ownerRef, "http://")
	s, ok := strings.CutPrefix(s, "https://")
	protocolIncluded = protocolIncluded || ok

	err := fmt.Errorf("owner reference %q invalid, valid format is [https://][<host>/]<name>", ownerRef)

	var o Owner
	parts := strings.Split(s, "/")
	switch len(parts) {
	case 2:
		o.host, o.name = parts[0], parts[1]
	case 1:
		if protocolIncluded || parts[0] == "" {
			return o, err
		}
		o.name = parts[0]
	default:
		return o, err
	}
	if o.host == "" {
		o.host = defaultOwnerHost
	}
	return o, nil
}

func (o Owner) String() string {
	return path.Join(o.host, o.name)
}