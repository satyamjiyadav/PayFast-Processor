package uid

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
)

// Generate creates a new ULID with an optional prefix (e.g., "pay_", "pm_")
// ULIDs are lexicographically sortable, making them great for primary keys.
func Generate(prefix string) string {
	t := time.Now()
	entropy := ulid.Monotonic(rand.Reader, 0)
	id := ulid.MustNew(ulid.Timestamp(t), entropy)
	
	if prefix == "" {
		return id.String()
	}
	return fmt.Sprintf("%s%s", prefix, id.String())
}
