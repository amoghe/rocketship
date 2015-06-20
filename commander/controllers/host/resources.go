package host

import (
	"encoding/json"
	"fmt"
)

type DHCPProfileResource struct {
	DNSMode            string // One of Mode[None|Append|Prepend|Supercede]
	OverrideHostname   bool   // Whether to supercede the name returned by the dhcp server
	OverrideDomainName bool   // Whether to supercede the name returned by the dhcp server

	RequireOptions []string // OptionsSeparator separated string
	RequestOptions []string // OptionsSeparator separated string
}

func (r *DHCPProfileResource) FromDHCPProfile(d DHCPProfile) error {
	var (
		deserializedRequestOpts = []string{}
		deserializedRequireOpts = []string{}
	)

	if len(d.RequestOptions) > 0 {
		err := json.Unmarshal([]byte(d.RequestOptions), &deserializedRequestOpts)
		if err != nil {
			return err
		}
	}
	if len(d.RequireOptions) > 0 {
		err := json.Unmarshal([]byte(d.RequireOptions), &deserializedRequireOpts)
		if err != nil {
			return err
		}
	}

	r.DNSMode = d.DNSMode
	r.OverrideHostname = d.OverrideHostname
	r.OverrideDomainName = d.OverrideDomainName
	r.RequestOptions = deserializedRequestOpts
	r.RequireOptions = deserializedRequireOpts

	return nil
}

func (r *DHCPProfileResource) ToDHCPProfile() (DHCPProfile, error) {

	serializedRequestOpts, err := json.Marshal(r.RequestOptions)
	if err != nil {
		return DHCPProfile{}, err
	}
	serializedRequireOpts, err := json.Marshal(r.RequireOptions)
	if err != nil {
		return DHCPProfile{}, err
	}
	if r.DNSMode != ModeAppend && r.DNSMode != ModePrepend && r.DNSMode != ModeOverride {
		return DHCPProfile{}, fmt.Errorf("Invalid DNSMode")
	}

	return DHCPProfile{
		DNSMode:            r.DNSMode,
		OverrideHostname:   r.OverrideHostname,
		OverrideDomainName: r.OverrideDomainName,
		RequireOptions:     string(serializedRequireOpts),
		RequestOptions:     string(serializedRequestOpts),

		//AppendOptions:    "{}",
		//PrependOptions:   "{}",
		//SupercedeOptions: "{}",
	}, nil
}
