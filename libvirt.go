package nestedvirt

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"

	libvirt "github.com/digitalocean/go-libvirt"
)

const novaMetadataNamespace = "http://openstack.org/xmlns/libvirt/nova/1.1"

type libvirtClient struct {
	conn *libvirt.Libvirt
}

func openLibvirt(uri string) (*libvirtClient, error) {
	if uri == "" {
		return nil, fmt.Errorf("libvirt URI: empty")
	}

	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	conn, err := libvirt.ConnectToURI(parsed)
	if err != nil {
		return nil, err
	}

	return &libvirtClient{conn: conn}, nil
}

func (c *libvirtClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}

	return c.conn.Disconnect()
}

func (c *libvirtClient) Domain(vm *VMIdentity, pid int) (*LibvirtDomain, []FindingError) {
	if vm == nil {
		return nil, nil
	}

	domain, err := c.lookupDomain(vm)
	if err != nil {
		return nil, []FindingError{processError(pid, "libvirt_lookup_domain", err)}
	}

	result := &LibvirtDomain{
		Name: domain.Name,
		UUID: formatLibvirtUUID(domain.UUID),
	}

	metadata, err := c.conn.DomainGetMetadata(
		*domain,
		int32(libvirt.DomainMetadataElement),
		libvirt.OptString{novaMetadataNamespace},
		libvirt.DomainAffectCurrent,
	)
	if err == nil {
		nova, parseErr := parseNovaMetadata(metadata)
		if parseErr != nil {
			return result, []FindingError{processError(pid, "libvirt_nova_metadata_parse", parseErr)}
		}
		result.NovaMetadata = nova
		return result, nil
	}

	if isNoDomainMetadata(err) {
		return result, nil
	}

	return result, []FindingError{processError(pid, "libvirt_nova_metadata", err)}
}

func (c *libvirtClient) lookupDomain(vm *VMIdentity) (*libvirt.Domain, error) {
	var errs []error

	if vm.UUID != "" {
		uuid, err := parseLibvirtUUID(vm.UUID)
		if err != nil {
			errs = append(errs, fmt.Errorf("uuid %q: %w", vm.UUID, err))
		} else if domain, err := c.conn.DomainLookupByUUID(uuid); err == nil {
			return &domain, nil
		} else {
			errs = append(errs, fmt.Errorf("uuid %q: %w", vm.UUID, err))
		}
	}

	if vm.Name != "" {
		domain, err := c.conn.DomainLookupByName(vm.Name)
		if err == nil {
			return &domain, nil
		}
		errs = append(errs, fmt.Errorf("name %q: %w", vm.Name, err))
	}

	if len(errs) == 0 {
		errs = append(errs, fmt.Errorf("missing libvirt domain identity"))
	}

	return nil, errors.Join(errs...)
}

func isNoDomainMetadata(err error) bool {
	var libvirtErr libvirt.Error
	if errors.As(err, &libvirtErr) {
		return libvirtErr.Code == uint32(libvirt.ErrNoDomainMetadata)
	}
	return false
}

func parseLibvirtUUID(value string) (libvirt.UUID, error) {
	var uuid libvirt.UUID

	normalized := strings.ReplaceAll(strings.TrimSpace(value), "-", "")
	decoded, err := hex.DecodeString(normalized)
	if err != nil {
		return uuid, err
	}
	if len(decoded) != len(uuid) {
		return uuid, fmt.Errorf("invalid UUID length %d", len(decoded))
	}

	copy(uuid[:], decoded)
	return uuid, nil
}

func formatLibvirtUUID(uuid libvirt.UUID) string {
	encoded := hex.EncodeToString(uuid[:])
	return encoded[0:8] + "-" +
		encoded[8:12] + "-" +
		encoded[12:16] + "-" +
		encoded[16:20] + "-" +
		encoded[20:32]
}
