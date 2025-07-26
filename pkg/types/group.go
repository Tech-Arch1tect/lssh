package types

type Group struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Hosts       []*Host  `json:"hosts"`
	SubGroups   []*Group `json:"subgroups,omitempty"`
}

func (g *Group) AllHosts() []*Host {
	var allHosts []*Host
	allHosts = append(allHosts, g.Hosts...)

	for _, subGroup := range g.SubGroups {
		allHosts = append(allHosts, subGroup.AllHosts()...)
	}

	return allHosts
}
