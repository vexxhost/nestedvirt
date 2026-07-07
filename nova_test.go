package nestedvirt

import "testing"

func TestParseNovaMetadata(t *testing.T) {
	raw := `<nova:instance xmlns:nova="http://openstack.org/xmlns/libvirt/nova/1.1">
  <nova:package version="26.2.1"/>
  <nova:name>customer-facing-name</nova:name>
  <nova:hostname>vm-hostname</nova:hostname>
  <nova:creationTime>2026-07-07T01:02:03Z</nova:creationTime>
  <nova:flavor name="vh.8c32r">
    <nova:memory>32768</nova:memory>
    <nova:disk>160</nova:disk>
    <nova:swap>0</nova:swap>
    <nova:ephemeral>0</nova:ephemeral>
    <nova:vcpus>8</nova:vcpus>
  </nova:flavor>
  <nova:owner>
    <nova:user uuid="user-uuid">demo-user</nova:user>
    <nova:project uuid="project-uuid">demo-project</nova:project>
  </nova:owner>
  <nova:root type="image" uuid="image-uuid"/>
</nova:instance>`

	got, err := parseNovaMetadata(raw)
	if err != nil {
		t.Fatal(err)
	}

	if got.Namespace != novaMetadataNamespace {
		t.Fatalf("Namespace = %q, want %q", got.Namespace, novaMetadataNamespace)
	}
	if got.Name != "customer-facing-name" {
		t.Fatalf("Name = %q, want customer-facing-name", got.Name)
	}
	if got.Hostname != "vm-hostname" {
		t.Fatalf("Hostname = %q, want vm-hostname", got.Hostname)
	}
	if got.PackageVersion != "26.2.1" {
		t.Fatalf("PackageVersion = %q, want 26.2.1", got.PackageVersion)
	}
	if got.Flavor == nil {
		t.Fatal("Flavor is nil")
	}
	if got.Flavor.Name != "vh.8c32r" || got.Flavor.VCPUs != "8" || got.Flavor.MemoryMiB != "32768" {
		t.Fatalf("Flavor = %#v, want parsed flavor", got.Flavor)
	}
	if got.Owner == nil {
		t.Fatal("Owner is nil")
	}
	if got.Owner.User.UUID != "user-uuid" || got.Owner.Project.UUID != "project-uuid" {
		t.Fatalf("Owner = %#v, want parsed owner UUIDs", got.Owner)
	}
	if got.Root == nil || got.Root.Type != "image" || got.Root.UUID != "image-uuid" {
		t.Fatalf("Root = %#v, want parsed image root", got.Root)
	}
	if got.RawXML != raw {
		t.Fatal("RawXML was not preserved")
	}
}
