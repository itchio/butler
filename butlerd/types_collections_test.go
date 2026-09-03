package butlerd

import (
	"testing"

	itchio "github.com/itchio/go-itchio"
	"github.com/stretchr/testify/require"
)

func TestCollectionsParamsLayoutValidation(t *testing.T) {
	require.NoError(t, CollectionsCreateParams{ProfileID: 1}.Validate())
	require.NoError(t, CollectionsCreateParams{ProfileID: 1, Layout: itchio.CollectionLayoutList}.Validate())
	require.Error(t, CollectionsCreateParams{ProfileID: 1, Layout: "mosaic"}.Validate())
	require.Error(t, CollectionsCreateParams{}.Validate())

	grid := itchio.CollectionLayoutGrid
	bogus := itchio.CollectionLayout("mosaic")
	require.NoError(t, CollectionsUpdateParams{ProfileID: 1, CollectionID: 2}.Validate())
	require.NoError(t, CollectionsUpdateParams{ProfileID: 1, CollectionID: 2, Layout: &grid}.Validate())
	require.Error(t, CollectionsUpdateParams{ProfileID: 1, CollectionID: 2, Layout: &bogus}.Validate())
	require.Error(t, CollectionsUpdateParams{ProfileID: 1}.Validate())

	require.NoError(t, CollectionsOrderGamesParams{ProfileID: 1, CollectionID: 2, GameIDs: []int64{}}.Validate())
	require.Error(t, CollectionsOrderGamesParams{ProfileID: 1, CollectionID: 2, GameIDs: make([]int64, 501)}.Validate())
}
