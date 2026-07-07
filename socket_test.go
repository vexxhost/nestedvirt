package nestedvirt

import "testing"

func TestLibvirtDomainNameFromMonitorSocket(t *testing.T) {
	tests := map[string]struct {
		path string
		want string
	}{
		"libvirt qemu monitor": {
			path: "/var/lib/libvirt/qemu/domain-15424-instance-0262a1f1/monitor.sock",
			want: "instance-0262a1f1",
		},
		"name with hyphen": {
			path: "/var/lib/libvirt/qemu/domain-7-prod-api-1/qmp.sock",
			want: "prod-api-1",
		},
		"not libvirt domain directory": {
			path: "/run/qemu/example/monitor.sock",
			want: "",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := libvirtDomainNameFromMonitorSocket(test.path); got != test.want {
				t.Fatalf("libvirtDomainNameFromMonitorSocket() = %q, want %q", got, test.want)
			}
		})
	}
}
