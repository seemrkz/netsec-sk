package enrich

import (
	"context"
	"errors"
	"net"
	"time"
)

var ErrNotFound = errors.New("rdns_not_found")

type ReverseDNS struct {
	IP            string
	PTRName       string
	Status        string
	LookedUpAtUTC string
}

type LookupFunc func(ctx context.Context, ip string) (string, error)

type Options struct {
	Enabled     bool
	IsNewDevice bool
	MgmtIP      string
	Now         time.Time
	Lookup      LookupFunc
}

func MaybeLookup(opts Options) (ReverseDNS, bool) {
	if !opts.Enabled || !opts.IsNewDevice || !isIPv4(opts.MgmtIP) || opts.Lookup == nil {
		return ReverseDNS{}, false
	}

	const attempts = 2
	lastStatus := "error"
	lastPtr := ""
	for i := 0; i < attempts; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		ptr, err := opts.Lookup(ctx, opts.MgmtIP)
		cancel()

		switch {
		case err == nil && ptr != "":
			lastStatus = "ok"
			lastPtr = ptr
			i = attempts
		case errors.Is(err, ErrNotFound):
			lastStatus = "not_found"
			i = attempts
		case errors.Is(err, context.DeadlineExceeded):
			lastStatus = "timeout"
		default:
			lastStatus = "error"
		}
	}

	return ReverseDNS{
		IP:            opts.MgmtIP,
		PTRName:       lastPtr,
		Status:        lastStatus,
		LookedUpAtUTC: opts.Now.UTC().Format(time.RFC3339),
	}, true
}

func isIPv4(v string) bool {
	ip := net.ParseIP(v)
	return ip != nil && ip.To4() != nil
}
