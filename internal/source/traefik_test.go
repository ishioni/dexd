package source

import (
	"sort"
	"testing"
)

type endpointWant struct {
	DNSName    string
	Target     string
	RecordType string
	Resource   string
}

func TestEndpointsFromLabels(t *testing.T) {
	const defaultTarget = "10.1.2.241"
	const ownerID = "test-owner"

	tests := []struct {
		name      string
		container string
		labels    map[string]string
		want      []endpointWant
	}{
		{
			name:      "no labels",
			container: "c1",
			labels:    map[string]string{},
		},
		{
			name:      "traefik enable alone is ignored",
			container: "c1",
			labels: map[string]string{
				"traefik.enable":                "true",
				"traefik.http.routers.foo.rule": "Host(`foo.example.com`)",
			},
		},
		{
			name:      "single host",
			container: "c1",
			labels: map[string]string{
				"dexd.enabled":                  "true",
				"traefik.http.routers.foo.rule": "Host(`foo.example.com`)",
			},
			want: []endpointWant{{DNSName: "foo.example.com", Target: defaultTarget, RecordType: "A"}},
		},
		{
			name:      "enabled without router rules has no endpoints",
			container: "c1",
			labels:    map[string]string{"dexd.enabled": "true"},
		},
		{
			name:      "multiple hosts",
			container: "c1",
			labels: map[string]string{
				"dexd.enabled":                  "true",
				"traefik.http.routers.foo.rule": "Host(`a.example.com`) || Host(`b.example.com`)",
			},
			want: []endpointWant{
				{DNSName: "a.example.com", Target: defaultTarget, RecordType: "A"},
				{DNSName: "b.example.com", Target: defaultTarget, RecordType: "A"},
			},
		},
		{
			name:      "host regexp is skipped",
			container: "c1",
			labels: map[string]string{
				"dexd.enabled":                  "true",
				"traefik.http.routers.foo.rule": "HostRegexp(`^.+\\.example\\.com$`) || Host(`a.example.com`)",
			},
			want: []endpointWant{{DNSName: "a.example.com", Target: defaultTarget, RecordType: "A"}},
		},
		{
			name:      "unresolved hostname is skipped",
			container: "c1",
			labels: map[string]string{
				"dexd.enabled":                  "true",
				"traefik.http.routers.foo.rule": "Host(`${HOST}`) || Host(`real.example.com`)",
			},
			want: []endpointWant{{DNSName: "real.example.com", Target: defaultTarget, RecordType: "A"}},
		},
		{
			name:      "multiple routers",
			container: "c1",
			labels: map[string]string{
				"dexd.enabled":                  "true",
				"traefik.http.routers.foo.rule": "Host(`foo.example.com`)",
				"traefik.http.routers.bar.rule": "Host(`bar.example.com`)",
			},
			want: []endpointWant{
				{DNSName: "bar.example.com", Target: defaultTarget, RecordType: "A"},
				{DNSName: "foo.example.com", Target: defaultTarget, RecordType: "A"},
			},
		},
		{
			name:      "disabled container is ignored",
			container: "c1",
			labels: map[string]string{
				"dexd.enabled":                  "false",
				"traefik.http.routers.foo.rule": "Host(`foo.example.com`)",
			},
		},
		{
			name:      "container target overrides default",
			container: "c1",
			labels: map[string]string{
				"dexd.enabled":                  "true",
				"dexd.target":                   "10.9.8.7",
				"traefik.http.routers.foo.rule": "Host(`foo.example.com`)",
			},
			want: []endpointWant{{DNSName: "foo.example.com", Target: "10.9.8.7", RecordType: "A"}},
		},
		{
			name:      "hostname target is cname",
			container: "c1",
			labels: map[string]string{
				"dexd.enabled":                  "true",
				"dexd.target":                   "traefik.example.com",
				"traefik.http.routers.foo.rule": "Host(`foo.example.com`)",
			},
			want: []endpointWant{{DNSName: "foo.example.com", Target: "traefik.example.com", RecordType: "CNAME"}},
		},
		{
			name:      "router target overrides container target",
			container: "c1",
			labels: map[string]string{
				"dexd.enabled":                  "true",
				"dexd.target":                   "10.9.8.7",
				"dexd.routers.foo.target":       "10.1.1.9",
				"traefik.http.routers.foo.rule": "Host(`foo.example.com`)",
				"traefik.http.routers.bar.rule": "Host(`bar.example.com`)",
			},
			want: []endpointWant{
				{DNSName: "bar.example.com", Target: "10.9.8.7", RecordType: "A"},
				{DNSName: "foo.example.com", Target: "10.1.1.9", RecordType: "A"},
			},
		},
		{
			name:      "dotted router name supports target override",
			container: "c1",
			labels: map[string]string{
				"dexd.enabled":                      "true",
				"dexd.routers.foo.bar.target":       "10.1.1.9",
				"traefik.http.routers.foo.bar.rule": "Host(`foo.example.com`)",
			},
			want: []endpointWant{{DNSName: "foo.example.com", Target: "10.1.1.9", RecordType: "A"}},
		},
		{
			name:      "router cname target",
			container: "c1",
			labels: map[string]string{
				"dexd.enabled":                  "true",
				"dexd.routers.foo.target":       "traefik.example.com",
				"traefik.http.routers.foo.rule": "Host(`foo.example.com`)",
				"traefik.http.routers.bar.rule": "Host(`bar.example.com`)",
			},
			want: []endpointWant{
				{DNSName: "bar.example.com", Target: defaultTarget, RecordType: "A"},
				{DNSName: "foo.example.com", Target: "traefik.example.com", RecordType: "CNAME"},
			},
		},
		{
			name:      "router skip",
			container: "c1",
			labels: map[string]string{
				"dexd.enabled":                  "true",
				"dexd.routers.foo.skip":         "true",
				"traefik.http.routers.foo.rule": "Host(`foo.example.com`)",
				"traefik.http.routers.bar.rule": "Host(`bar.example.com`)",
			},
			want: []endpointWant{{DNSName: "bar.example.com", Target: defaultTarget, RecordType: "A"}},
		},
		{
			name:      "extra hostnames append to parsed hosts",
			container: "rustfs",
			labels: map[string]string{
				"dexd.enabled":                        "true",
				"dexd.routers.rustfs.extra-hostnames": "*.rs.example.com",
				"traefik.http.routers.rustfs.rule":    "Host(`rs.example.com`) || HostRegexp(`^.+\\.rs\\.example\\.com$`)",
			},
			want: []endpointWant{
				{DNSName: "*.rs.example.com", Target: defaultTarget, RecordType: "A"},
				{DNSName: "rs.example.com", Target: defaultTarget, RecordType: "A"},
			},
		},
		{
			name:      "hostnames override parsed hosts",
			container: "rustfs",
			labels: map[string]string{
				"dexd.enabled":                     "true",
				"dexd.routers.rustfs.hostnames":    "wanted.example.com,*.wanted.example.com",
				"traefik.http.routers.rustfs.rule": "Host(`ignored.example.com`)",
			},
			want: []endpointWant{
				{DNSName: "*.wanted.example.com", Target: defaultTarget, RecordType: "A"},
				{DNSName: "wanted.example.com", Target: defaultTarget, RecordType: "A"},
			},
		},
		{
			name:      "router skip ignores overridden hostnames",
			container: "rustfs",
			labels: map[string]string{
				"dexd.enabled":                     "true",
				"dexd.routers.rustfs.hostnames":    "wanted.example.com,*.wanted.example.com",
				"dexd.routers.rustfs.skip":         "true",
				"traefik.http.routers.rustfs.rule": "Host(`ignored.example.com`)",
			},
		},
		{
			name:      "standalone host block uses default target",
			container: "traefik",
			labels: map[string]string{
				"dexd.enabled":                   "true",
				"dexd.hosts.dashboard.hostnames": "traefik.example.com",
			},
			want: []endpointWant{{DNSName: "traefik.example.com", Target: defaultTarget, RecordType: "A", Resource: "docker/traefik/hosts/dashboard"}},
		},
		{
			name:      "standalone host block target overrides container and default",
			container: "traefik",
			labels: map[string]string{
				"dexd.enabled":                   "true",
				"dexd.target":                    "10.9.8.7",
				"dexd.hosts.dashboard.hostnames": "traefik.example.com,*.traefik.example.com",
				"dexd.hosts.dashboard.target":    "traefik.internal.example.com",
			},
			want: []endpointWant{
				{DNSName: "*.traefik.example.com", Target: "traefik.internal.example.com", RecordType: "CNAME", Resource: "docker/traefik/hosts/dashboard"},
				{DNSName: "traefik.example.com", Target: "traefik.internal.example.com", RecordType: "CNAME", Resource: "docker/traefik/hosts/dashboard"},
			},
		},
		{
			name:      "standalone host block skip",
			container: "traefik",
			labels: map[string]string{
				"dexd.enabled":                   "true",
				"dexd.hosts.dashboard.hostnames": "traefik.example.com,*.traefik.example.com",
				"dexd.hosts.dashboard.skip":      "true",
			},
		},
		{
			name:      "mixed routers use independent targets",
			container: "rustfs",
			labels: map[string]string{
				"dexd.enabled":                      "true",
				"dexd.routers.console.target":       "traefik.example.com",
				"traefik.http.routers.s3.rule":      "Host(`bucket.example.com`)",
				"traefik.http.routers.console.rule": "Host(`console.example.com`)",
			},
			want: []endpointWant{
				{DNSName: "bucket.example.com", Target: defaultTarget, RecordType: "A"},
				{DNSName: "console.example.com", Target: "traefik.example.com", RecordType: "CNAME"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EndpointsFromLabels(tt.container, tt.labels, defaultTarget, ownerID)
			gotEndpoints := make([]endpointWant, len(got))
			for i, ep := range got {
				gotEndpoints[i] = endpointWant{DNSName: ep.DNSName, Target: ep.Target, RecordType: ep.RecordType, Resource: ep.Resource}
				if ep.OwnerID != ownerID {
					t.Errorf("endpoint %s: OwnerID = %q, want %q", ep.DNSName, ep.OwnerID, ownerID)
				}
			}
			for i := range tt.want {
				if tt.want[i].Resource == "" {
					tt.want[i].Resource = "docker/" + tt.container
				}
			}
			sortEndpoints(gotEndpoints)
			sortEndpoints(tt.want)
			if !equalEndpoints(gotEndpoints, tt.want) {
				t.Errorf("Endpoints = %v, want %v", gotEndpoints, tt.want)
			}
		})
	}
}

func sortEndpoints(eps []endpointWant) {
	sort.Slice(eps, func(i, j int) bool {
		if eps[i].DNSName != eps[j].DNSName {
			return eps[i].DNSName < eps[j].DNSName
		}
		return eps[i].RecordType < eps[j].RecordType
	})
}

func equalEndpoints(a, b []endpointWant) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestDetectRecordType(t *testing.T) {
	tests := []struct {
		target string
		want   string
	}{
		{"10.0.0.1", "A"},
		{"192.168.1.254", "A"},
		{"10.1.2.241", "A"},
		{"traefik.example.com", "CNAME"},
		{"my-host.internal", "CNAME"},
		{"::1", "CNAME"},
		{"2001:db8::1", "CNAME"},
		{"", "CNAME"},
		{"not-an-ip.foo", "CNAME"},
	}
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			if got := detectRecordType(tt.target); got != tt.want {
				t.Errorf("detectRecordType(%q) = %q, want %q", tt.target, got, tt.want)
			}
		})
	}
}
