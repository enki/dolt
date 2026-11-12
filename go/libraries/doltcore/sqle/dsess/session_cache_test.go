// Copyright 2026 Dolthub, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dsess

import (
	"testing"

	"github.com/dolthub/go-mysql-server/sql"
	"github.com/stretchr/testify/require"

	"github.com/dolthub/dolt/go/libraries/doltcore/doltdb"
	"github.com/dolthub/dolt/go/store/hash"
)

// Views are cached per schema and dolt_schemas hash: once a schema's views are cached, a name the cache lacks is not a
// view of that schema, while another schema or another hash is not cached at all.
func TestSessionCacheViews(t *testing.T) {
	c := newSessionCache()
	key := doltdb.DataCacheKey{Hash: hash.Of([]byte("schemas v1"))}
	other := doltdb.DataCacheKey{Hash: hash.Of([]byte("schemas v2"))}
	require.False(t, c.ViewsCached(key, ""))

	c.CacheViews(key, []sql.ViewDefinition{{Name: "MyView", CreateViewStatement: "create view MyView as select 1"}}, "")
	require.True(t, c.ViewsCached(key, ""))
	require.False(t, c.ViewsCached(key, "s2"))
	require.False(t, c.ViewsCached(other, ""))

	view, ok := c.GetCachedViewDefinition(key, TableCacheKey{Name: "myview"})
	require.True(t, ok)
	require.Equal(t, "MyView", view.Name)
	_, ok = c.GetCachedViewDefinition(key, TableCacheKey{Name: "mytable"})
	require.False(t, ok)
	_, ok = c.GetCachedViewDefinition(key, TableCacheKey{Name: "myview", Schema: "s2"})
	require.False(t, ok)

	// A schema without views is cached as such.
	c.CacheViews(key, nil, "S2")
	require.True(t, c.ViewsCached(key, "s2"))
	_, ok = c.GetCachedViewDefinition(key, TableCacheKey{Name: "myview", Schema: "s2"})
	require.False(t, ok)
}

// Caching a key's triggers again replaces them, and no triggers is a cached answer distinct from none cached.
func TestSessionCacheTriggers(t *testing.T) {
	c := newSessionCache()
	key := doltdb.DataCacheKey{Hash: hash.Of([]byte("schemas v1"))}
	_, ok := c.GetCachedTriggers(key, "")
	require.False(t, ok)

	c.CacheTriggers(key, nil, "")
	triggers, ok := c.GetCachedTriggers(key, "")
	require.True(t, ok)
	require.Empty(t, triggers)

	one := []sql.TriggerDefinition{{Name: "t_ai", CreateStatement: "create trigger t_ai after insert on t for each row set @x = 1"}}
	c.CacheTriggers(key, one, "")
	c.CacheTriggers(key, one, "")
	triggers, ok = c.GetCachedTriggers(key, "")
	require.True(t, ok)
	require.Equal(t, one, triggers)
}
