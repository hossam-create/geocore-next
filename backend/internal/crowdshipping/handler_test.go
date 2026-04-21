package crowdshipping

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPaginateMatchResults_StableAcrossPages(t *testing.T) {
	matches := make([]MatchResult, 0, 8)
	for i := 0; i < 8; i++ {
		matches = append(matches, MatchResult{MatchScore: float64(100 - i), EstimatedDelivery: fmt.Sprintf("d-%d", i)})
	}

	p1, page1, per1 := paginateMatchResults(matches, 1, 3)
	require.Equal(t, 1, page1)
	require.Equal(t, 3, per1)
	require.Len(t, p1, 3)
	require.Equal(t, 100.0, p1[0].MatchScore)
	require.Equal(t, 99.0, p1[1].MatchScore)
	require.Equal(t, 98.0, p1[2].MatchScore)

	p2, page2, per2 := paginateMatchResults(matches, 2, 3)
	require.Equal(t, 2, page2)
	require.Equal(t, 3, per2)
	require.Len(t, p2, 3)
	require.Equal(t, 97.0, p2[0].MatchScore)
	require.Equal(t, 96.0, p2[1].MatchScore)
	require.Equal(t, 95.0, p2[2].MatchScore)
}

func TestTravelerMatchesCacheTTL(t *testing.T) {
	key := "find_travelers:test"
	setCachedTravelerMatches(key, []MatchResult{{MatchScore: 88.5}}, 50*time.Millisecond)

	cached, hit := getCachedTravelerMatches(key)
	require.True(t, hit)
	require.Len(t, cached, 1)
	require.Equal(t, 88.5, cached[0].MatchScore)

	time.Sleep(70 * time.Millisecond)
	_, hit = getCachedTravelerMatches(key)
	require.False(t, hit)
}
