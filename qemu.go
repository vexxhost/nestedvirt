package nestedvirt

import "strings"

func qemuVMIdentity(args []string) *VMIdentity {
	var identity VMIdentity

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "-uuid" && i+1 < len(args):
			setIdentityValue(&identity.UUID, args[i+1], &identity.Sources, "-uuid")
			i++
		case strings.HasPrefix(arg, "-uuid="):
			setIdentityValue(&identity.UUID, strings.TrimPrefix(arg, "-uuid="), &identity.Sources, "-uuid")
		case arg == "-name" && i+1 < len(args):
			setIdentityValue(&identity.Name, qemuGuestName(args[i+1]), &identity.Sources, "-name")
			i++
		case strings.HasPrefix(arg, "-name="):
			setIdentityValue(&identity.Name, qemuGuestName(strings.TrimPrefix(arg, "-name=")), &identity.Sources, "-name")
		case arg == "-smbios" && i+1 < len(args):
			setIdentityValue(&identity.UUID, qemuOptionValue(args[i+1], "uuid"), &identity.Sources, "-smbios")
			i++
		case strings.HasPrefix(arg, "-smbios="):
			setIdentityValue(&identity.UUID, qemuOptionValue(strings.TrimPrefix(arg, "-smbios="), "uuid"), &identity.Sources, "-smbios")
		}
	}

	if identity.Name == "" && identity.UUID == "" {
		return nil
	}

	return &identity
}

func withVMName(identity *VMIdentity, name, source string) *VMIdentity {
	if strings.TrimSpace(name) == "" {
		return identity
	}
	if identity == nil {
		identity = &VMIdentity{}
	}

	setIdentityValue(&identity.Name, name, &identity.Sources, source)
	return identity
}

func qemuGuestName(value string) string {
	if guest := qemuOptionValue(value, "guest"); guest != "" {
		return guest
	}

	if name := qemuOptionValue(value, "name"); name != "" {
		return name
	}

	first, _, _ := strings.Cut(value, ",")
	return strings.TrimSpace(first)
}

func qemuOptionValue(value, key string) string {
	for _, field := range strings.Split(value, ",") {
		name, candidate, ok := strings.Cut(field, "=")
		if !ok {
			continue
		}
		if strings.TrimSpace(name) == key {
			return strings.TrimSpace(candidate)
		}
	}

	return ""
}

func setIdentityValue(dst *string, value string, sources *[]string, source string) {
	value = strings.TrimSpace(value)
	if value == "" || *dst != "" {
		return
	}

	*dst = value
	*sources = append(*sources, source)
}
