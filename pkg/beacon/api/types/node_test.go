package types_test

import (
	"testing"

	"github.com/ethpandaops/beacon/pkg/beacon/api/types"
	"github.com/stretchr/testify/require"
)

func TestIdentity_GetEnode(t *testing.T) {
	identity := &types.Identity{
		ENR: "enr:-IS4QHCYrYZbAKWCBRlAy5zzaDZXJBGkcnh4MHcBFZntXNFrdvJjX04jRzjzCBOonrkTfj499SZuOh8R33Ls8RRcy5wBgmlkgnY0gmlwhH8AAAGJc2VjcDI1NmsxoQPKY0yuDUmstAHYpMa2_oxVtw0RW_QAdpzBQA8yWM0xOIN1ZHCCdl8",
	}

	enode, err := identity.GetEnode()
	require.NoError(t, err)
	require.NotNil(t, enode)

	// Verify enode details
	require.Equal(t, "127.0.0.1", enode.IP().String())
	require.Equal(t, 30303, enode.UDP())
	require.Equal(t, 0, enode.TCP())
}
