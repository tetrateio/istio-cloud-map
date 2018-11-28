package infer

import (
	"fmt"
	"net"
	"strings"

	"istio.io/api/networking/v1alpha3"
	"istio.io/istio/pilot/pkg/model"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceEntry infers an Istio service entry based on provided information
// TODO: Owners, needs Istio upstreaming!
// TODO: Namespaces...
func ServiceEntry(owner v1.OwnerReference, host string, endpoints []*v1alpha3.ServiceEntry_Endpoint) model.Config {
	addresses := []string{}
	if len(endpoints) > 0 {
		if ip := net.ParseIP(endpoints[0].Address); ip != nil {
			addresses = []string{endpoints[0].Address}
		}
	}
	return model.Config{
		ConfigMeta: model.ConfigMeta{
			Name: ServiceEntryName(host),
			Type: model.ServiceEntry.Type,
			// This concatenation is to placate weird Istio behaviour; not sure if bug with Istio code?
			Group:   model.ServiceEntry.Group + model.IstioAPIGroupDomain,
			Version: model.ServiceEntry.Version,
			Domain:  model.IstioAPIGroupDomain,
		},
		Spec: &v1alpha3.ServiceEntry{
			Hosts:     []string{host},
			Addresses: addresses,
			// assume external for now
			Location:   v1alpha3.ServiceEntry_MESH_EXTERNAL,
			Resolution: Resolution(endpoints),
			Ports:      Ports(endpoints),
			Endpoints:  endpoints,
		},
	}
}

// Endpoint creates a Service Entry endpoint from an address and port
// It infers the port name from the port number
func Endpoint(address string, port uint32) *v1alpha3.ServiceEntry_Endpoint {
	return &v1alpha3.ServiceEntry_Endpoint{
		Address: address,
		Ports:   map[string]uint32{Proto(port): port},
	}
}

// Proto infers the port name based on the port number
func Proto(port uint32) string {
	switch port {
	case 80:
		return "http"
	case 443:
		return "https"
	default:
		return "tcp"
	}
}

// Ports uses a slice of Service Entry endpoints to create a de-duped slice of Istio Ports
// Infering name and protocol from the port number
func Ports(endpoints []*v1alpha3.ServiceEntry_Endpoint) []*v1alpha3.Port {
	dedup := map[uint32]*v1alpha3.Port{}
	for _, ep := range endpoints {
		for _, port := range ep.Ports {
			dedup[port] = &v1alpha3.Port{
				Name:     Proto(port),
				Number:   uint32(port),
				Protocol: strings.ToUpper(Proto(port)),
			}
		}
	}
	res := []*v1alpha3.Port{}
	for _, port := range dedup {
		res = append(res, port)
	}
	return res
}

// Resolution infers STATIC resolution if there are endpoints
// If there are no endpoints it infers DNS; otherwise will return STATIC
// TODO: this will probably need to be changed when we support non-IP based addresses
func Resolution(endpoints []*v1alpha3.ServiceEntry_Endpoint) v1alpha3.ServiceEntry_Resolution {
	if len(endpoints) == 0 {
		return v1alpha3.ServiceEntry_DNS
	}
	for _, ep := range endpoints {
		if addr := net.ParseIP(ep.Address); addr == nil {
			return v1alpha3.ServiceEntry_DNS // is not IP so DNS
		}
	}
	return v1alpha3.ServiceEntry_STATIC
}

// ServiceEntryName returns the service entry name based on the specificed host
func ServiceEntryName(host string) string {
	return fmt.Sprintf("cloudmap-%v", host)
}