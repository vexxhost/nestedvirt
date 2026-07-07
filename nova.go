package nestedvirt

import (
	"encoding/xml"
	"strings"
)

type novaInstanceXML struct {
	XMLName      xml.Name       `xml:"instance"`
	Name         string         `xml:"name"`
	Hostname     string         `xml:"hostname"`
	CreationTime string         `xml:"creationTime"`
	Package      novaPackageXML `xml:"package"`
	Flavor       novaFlavorXML  `xml:"flavor"`
	Owner        novaOwnerXML   `xml:"owner"`
	Root         novaRootXML    `xml:"root"`
}

type novaPackageXML struct {
	Version string `xml:"version,attr"`
}

type novaFlavorXML struct {
	Name      string `xml:"name,attr"`
	Memory    string `xml:"memory"`
	Disk      string `xml:"disk"`
	Swap      string `xml:"swap"`
	Ephemeral string `xml:"ephemeral"`
	VCPUs     string `xml:"vcpus"`
}

type novaOwnerXML struct {
	User    novaIdentityXML `xml:"user"`
	Project novaIdentityXML `xml:"project"`
}

type novaIdentityXML struct {
	UUID string `xml:"uuid,attr"`
	Name string `xml:",chardata"`
}

type novaRootXML struct {
	Type string `xml:"type,attr"`
	UUID string `xml:"uuid,attr"`
}

func parseNovaMetadata(raw string) (*NovaMetadata, error) {
	var instance novaInstanceXML
	if err := xml.Unmarshal([]byte(raw), &instance); err != nil {
		return nil, err
	}

	metadata := &NovaMetadata{
		Namespace:      namespaceOrDefault(instance.XMLName.Space),
		Name:           strings.TrimSpace(instance.Name),
		Hostname:       strings.TrimSpace(instance.Hostname),
		CreationTime:   strings.TrimSpace(instance.CreationTime),
		PackageVersion: strings.TrimSpace(instance.Package.Version),
	}

	if hasNovaFlavor(instance.Flavor) {
		metadata.Flavor = &NovaFlavor{
			Name:         strings.TrimSpace(instance.Flavor.Name),
			MemoryMiB:    strings.TrimSpace(instance.Flavor.Memory),
			DiskGiB:      strings.TrimSpace(instance.Flavor.Disk),
			SwapMiB:      strings.TrimSpace(instance.Flavor.Swap),
			EphemeralGiB: strings.TrimSpace(instance.Flavor.Ephemeral),
			VCPUs:        strings.TrimSpace(instance.Flavor.VCPUs),
		}
	}

	if hasNovaOwner(instance.Owner) {
		metadata.Owner = &NovaOwner{
			User:    novaIdentity(instance.Owner.User),
			Project: novaIdentity(instance.Owner.Project),
		}
	}

	if instance.Root.Type != "" || instance.Root.UUID != "" {
		metadata.Root = &NovaRoot{
			Type: strings.TrimSpace(instance.Root.Type),
			UUID: strings.TrimSpace(instance.Root.UUID),
		}
	}

	return metadata, nil
}

func namespaceOrDefault(namespace string) string {
	if namespace != "" {
		return namespace
	}
	return novaMetadataNamespace
}

func hasNovaFlavor(flavor novaFlavorXML) bool {
	return flavor.Name != "" ||
		flavor.Memory != "" ||
		flavor.Disk != "" ||
		flavor.Swap != "" ||
		flavor.Ephemeral != "" ||
		flavor.VCPUs != ""
}

func hasNovaOwner(owner novaOwnerXML) bool {
	return owner.User.UUID != "" ||
		strings.TrimSpace(owner.User.Name) != "" ||
		owner.Project.UUID != "" ||
		strings.TrimSpace(owner.Project.Name) != ""
}

func novaIdentity(identity novaIdentityXML) NovaIdentity {
	return NovaIdentity{
		UUID: strings.TrimSpace(identity.UUID),
		Name: strings.TrimSpace(identity.Name),
	}
}
