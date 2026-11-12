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

package sqle

import (
	"context"
	"testing"

	"github.com/dolthub/go-mysql-server/sql"
	"github.com/stretchr/testify/require"

	"github.com/dolthub/dolt/go/libraries/doltcore/dtestutils"
	"github.com/dolthub/dolt/go/libraries/doltcore/table/editor"
)

// A trigger list that fails to load is an error every time it is asked for, never remembered as no triggers, which
// would let writes run without their triggers: here a stored trigger whose creation time can't be read fails the load.
func TestGetTriggersCachesOnlyALoadedList(t *testing.T) {
	ctx := context.Background()
	dEnv := dtestutils.CreateTestEnv()
	defer dEnv.Close()
	db, err := NewDatabase(ctx, "dolt", dEnv.DbData(ctx), editor.Options{})
	require.NoError(t, err)
	engine, sqlCtx, err := NewTestEngine(dEnv, ctx, db)
	require.NoError(t, err)
	run := func(q string) error {
		_, iter, _, err := engine.Query(sqlCtx, q)
		if err != nil {
			return err
		}
		_, err = sql.RowIterToRows(sqlCtx, iter)
		return err
	}
	for _, q := range []string{"create table t (x int primary key)", "create table log (x int primary key)",
		"create trigger t_ai after insert on t for each row insert into log values (new.x)"} {
		require.NoError(t, run(q))
	}
	triggers, err := db.GetTriggers(sqlCtx)
	require.NoError(t, err)
	require.Len(t, triggers, 1)

	schemas, err := getOrCreateDoltSchemasTable(sqlCtx, db)
	require.NoError(t, err)
	inserter := schemas.Inserter(sqlCtx)
	require.NoError(t, inserter.Insert(sqlCtx, sql.Row{"trigger", "bad", "create trigger bad after insert on log for each row set @x = 1",
		[]byte(`{"CreatedAt": "never"}`), "STRICT_TRANS_TABLES"}))
	require.NoError(t, inserter.Close(sqlCtx))
	for i := 0; i < 2; i++ {
		_, err = db.GetTriggers(sqlCtx)
		require.ErrorContains(t, err, "does not contain creation time")
	}
	require.NoError(t, db.dropFragFromSchemasTable(sqlCtx, "trigger", "bad", nil))
	triggers, err = db.GetTriggers(sqlCtx)
	require.NoError(t, err)
	require.Len(t, triggers, 1)
	require.Equal(t, "t_ai", triggers[0].Name)
}
