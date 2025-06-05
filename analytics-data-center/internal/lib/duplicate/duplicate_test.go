package duplicate

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemoveDuplicate(t *testing.T) {
	input := []int{1, 2, 2, 3, 1}
	clean, dup := RemoveDuplicate(input)

	require.Equal(t, []int{1, 2, 3}, clean)
	require.Equal(t, []int{2, 1}, dup)
}

func TestRemoveDuplicateColumns(t *testing.T) {
	cols := []models.Column{
		{Name: "id"},
		{Name: "name"},
		{Name: "id"},
		{Name: "age", Alias: "age_alias"},
		{Name: "age2", Alias: "age_alias"},
		{Name: "weight"},
	}

	clean, dups := RemoveDuplicateColumns(cols)

	require.Len(t, clean, 4)
	require.Equal(t, []string{"id", "age_alias"}, dups)
	require.Equal(t, "id", clean[0].Name)
	require.Equal(t, "name", clean[1].Name)
	require.Equal(t, "age", clean[2].Name)
	require.Equal(t, "weight", clean[3].Name)
}
