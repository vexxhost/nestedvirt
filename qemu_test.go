package nestedvirt

import (
	"reflect"
	"testing"
)

func TestQEMUVMIdentity(t *testing.T) {
	tests := map[string]struct {
		args []string
		want *VMIdentity
	}{
		"libvirt name and uuid": {
			args: []string{
				"/usr/libexec/qemu-kvm",
				"-name", "guest=instance-0000002a,debug-threads=on",
				"-uuid", "11112222-3333-4444-5555-666677778888",
			},
			want: &VMIdentity{
				Name:    "instance-0000002a",
				UUID:    "11112222-3333-4444-5555-666677778888",
				Sources: []string{"-name", "-uuid"},
			},
		},
		"equals form": {
			args: []string{
				"qemu-system-x86_64",
				"-name=guest=test-vm",
				"-uuid=aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
			},
			want: &VMIdentity{
				Name:    "test-vm",
				UUID:    "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
				Sources: []string{"-name", "-uuid"},
			},
		},
		"smbios uuid": {
			args: []string{
				"qemu-system-x86_64",
				"-smbios", "type=1,manufacturer=OpenStack Foundation,uuid=99999999-8888-7777-6666-555555555555",
			},
			want: &VMIdentity{
				UUID:    "99999999-8888-7777-6666-555555555555",
				Sources: []string{"-smbios"},
			},
		},
		"no identity": {
			args: []string{"qemu-system-x86_64", "-m", "4096"},
			want: nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := qemuVMIdentity(test.args)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("qemuVMIdentity() = %#v, want %#v", got, test.want)
			}
		})
	}
}
