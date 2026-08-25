package federationruntime

import "github.com/sergii/specview/internal/federation"

type RepositorySelection struct {
	Group    federation.RepositoryGroup
	Instance federation.SourcedInstance
}

func SelectHost(projection Projection, hostID string) (HostStatus, bool) {
	for _, host := range projection.Hosts {
		if host.HostID == hostID {
			return host, true
		}
	}
	return HostStatus{}, false
}

func RepositoriesForHost(projection Projection, hostID string) []RepositorySelection {
	result := make([]RepositorySelection, 0)
	for _, group := range projection.Federation.Repositories {
		for _, instance := range group.Instances {
			if instance.HostID != hostID {
				continue
			}
			result = append(result, RepositorySelection{Group: group, Instance: instance})
		}
	}
	return result
}

func SelectRepository(projection Projection, hostID, instanceID string) (HostStatus, RepositorySelection, bool) {
	host, ok := SelectHost(projection, hostID)
	if !ok {
		return HostStatus{}, RepositorySelection{}, false
	}
	for _, selection := range RepositoriesForHost(projection, hostID) {
		if selection.Instance.InstanceID == instanceID {
			return host, selection, true
		}
	}
	return HostStatus{}, RepositorySelection{}, false
}
