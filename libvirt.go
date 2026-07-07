package nestedvirt

import (
	"errors"
	"fmt"

	libvirt "libvirt.org/go/libvirt"
)

const novaMetadataNamespace = "http://openstack.org/xmlns/libvirt/nova/1.1"

type libvirtClient struct {
	conn *libvirt.Connect
}

func openLibvirt(uri string) (*libvirtClient, error) {
	if uri == "" {
		return nil, fmt.Errorf("libvirt URI: empty")
	}

	conn, err := libvirt.NewConnect(uri)
	if err != nil {
		return nil, err
	}

	return &libvirtClient{conn: conn}, nil
}

func (c *libvirtClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}

	_, err := c.conn.Close()
	return err
}

func (c *libvirtClient) Domain(vm *VMIdentity, pid int) (*LibvirtDomain, []FindingError) {
	if vm == nil {
		return nil, nil
	}

	domain, err := c.lookupDomain(vm)
	if err != nil {
		return nil, []FindingError{processError(pid, "libvirt_lookup_domain", err)}
	}
	defer domain.Free()

	result := &LibvirtDomain{}

	name, err := domain.GetName()
	if err != nil {
		return result, []FindingError{processError(pid, "libvirt_domain_name", err)}
	}
	result.Name = name

	uuid, err := domain.GetUUIDString()
	if err != nil {
		return result, []FindingError{processError(pid, "libvirt_domain_uuid", err)}
	}
	result.UUID = uuid

	metadata, err := domain.GetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, novaMetadataNamespace, libvirt.DOMAIN_AFFECT_CURRENT)
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
		domain, err := c.conn.LookupDomainByUUIDString(vm.UUID)
		if err == nil {
			return domain, nil
		}
		errs = append(errs, fmt.Errorf("uuid %q: %w", vm.UUID, err))
	}

	if vm.Name != "" {
		domain, err := c.conn.LookupDomainByName(vm.Name)
		if err == nil {
			return domain, nil
		}
		errs = append(errs, fmt.Errorf("name %q: %w", vm.Name, err))
	}

	return nil, errors.Join(errs...)
}

func isNoDomainMetadata(err error) bool {
	var libvirtErr libvirt.Error
	if errors.As(err, &libvirtErr) {
		return libvirtErr.Code == libvirt.ERR_NO_DOMAIN_METADATA
	}
	return false
}
