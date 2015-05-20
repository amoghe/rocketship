package fqdn

type Hostname struct {
	ID       int64 `json:"-"`
	Hostname string
}

type Domain struct {
	ID     int64 `json:"-"`
	Domain string
}
