package cluster

import "github.com/colonyos/colonies/pkg/cluster/config"

// The configuration types live in pkg/cluster/config so that clients can use
// them without importing this package, which embeds an etcd server. These
// aliases keep existing users of cluster.Config and cluster.Node unchanged.
type (
	Node   = config.Node
	Config = config.Config
)

// ConvertJSONToConfig parses a cluster configuration; see config.ConvertJSONToConfig.
func ConvertJSONToConfig(jsonString string) (*Config, error) {
	return config.ConvertJSONToConfig(jsonString)
}
