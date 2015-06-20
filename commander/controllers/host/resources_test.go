package host

import (
	. "gopkg.in/check.v1"
)

//
// Test Suite
//

type ResourcesTestSuite struct{}

func (r *ResourcesTestSuite) TestDHCPProfileToResourceConversion(c *C) {

	goodProfiles := []DHCPProfile{
		DHCPProfile{
			TimingOptions:    "{}", // Dont care
			SendOptions:      "{}", // Dont care
			RequireOptions:   "[]",
			RequestOptions:   "[]",
			DNSMode:          ModeAppend,
			OverrideHostname: true,
		},
		DHCPProfile{
			TimingOptions:    DefaultTimingOptionsJSON,
			SendOptions:      DefaultSendOptionsJSON,
			RequireOptions:   DefaultRequestOptionsJSON,
			DNSMode:          ModeAppend,
			OverrideHostname: false,
		},
	}

	badProfiles := []DHCPProfile{
		DHCPProfile{
			RequireOptions:   "[]",
			RequestOptions:   "asdf", // bad json
			DNSMode:          ModeAppend,
			OverrideHostname: true,
		},
		DHCPProfile{
			RequireOptions:   "qwer", // bad json
			RequestOptions:   "[]",
			DNSMode:          ModeAppend,
			OverrideHostname: true,
		},
	}

	for _, d := range goodProfiles {
		resource := DHCPProfileResource{}
		c.Assert(resource.FromDHCPProfile(d), IsNil)
		c.Assert(resource.DNSMode, Equals, d.DNSMode)
		c.Assert(resource.OverrideHostname, Equals, d.OverrideHostname)
		c.Assert(resource.OverrideDomainName, Equals, d.OverrideDomainName)
	}

	for _, d := range badProfiles {
		resource := DHCPProfileResource{}
		c.Assert(resource.FromDHCPProfile(d), Not(IsNil))
	}
}

func (r *ResourcesTestSuite) TestDHCPResourceToProfileConversion(c *C) {

	goodResoures := []DHCPProfileResource{
		{
			DNSMode:          ModeAppend,
			RequireOptions:   []string{"subnet-mask"},
			RequestOptions:   []string{"subnet-mask", "routers", "domain-name"},
			OverrideHostname: true,
		},
		{
			DNSMode:          ModePrepend,
			RequireOptions:   []string{},
			RequestOptions:   []string{},
			OverrideHostname: true,
		},
	}

	badResoures := []DHCPProfileResource{
		{
			DNSMode:          "", // missing mode
			RequireOptions:   []string{"subnet-mask"},
			RequestOptions:   []string{"subnet-mask", "routers", "domain-name"},
			OverrideHostname: true,
		},
	}

	for _, resource := range goodResoures {
		d, err := resource.ToDHCPProfile()
		c.Assert(err, IsNil)
		c.Assert(resource.DNSMode, Equals, d.DNSMode)
		c.Assert(resource.OverrideHostname, Equals, d.OverrideHostname)
		c.Assert(resource.OverrideDomainName, Equals, d.OverrideDomainName)
	}

	for _, resource := range badResoures {
		err, _ := resource.ToDHCPProfile()
		c.Assert(err, Not(IsNil))
	}
}
