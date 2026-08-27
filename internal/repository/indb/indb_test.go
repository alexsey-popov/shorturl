package indb

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alexsey-popov/shorturl/internal/model"
	errors2 "github.com/alexsey-popov/shorturl/pkg/errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rep := New(db)
	assert.NotNil(t, rep)
}

func TestPing(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer db.Close()

	rep := New(db)

	t.Run("success", func(t *testing.T) {
		mock.ExpectPing()
		err := rep.Ping()
		assert.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		mock.ExpectPing().WillReturnError(errors.New("ping error"))
		err := rep.Ping()
		assert.Error(t, err)
	})
}

func TestGet(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rep := New(db)
	uID := "uuid-1"
	pref := "pref1"
	orig := "https://example.com"
	userID := "user-1"
	isDeleted := false

	t.Run("success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "prefix", "original_url", "user_id", "is_deleted"}).
			AddRow(uID, pref, orig, userID, isDeleted)

		mock.ExpectQuery("SELECT id, prefix, original_url, user_id, is_deleted FROM urls where prefix = \\$1").
			WithArgs(pref).
			WillReturnRows(rows)

		url, err := rep.Get(pref)
		assert.NoError(t, err)
		assert.Equal(t, uID, url.UUID)
		assert.Equal(t, pref, url.Prefix)
		assert.Equal(t, orig, url.OriginalURL)
		assert.Equal(t, userID, url.UserID)
		assert.Equal(t, isDeleted, url.IsDeleted)
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, prefix, original_url, user_id, is_deleted FROM urls where prefix = \\$1").
			WithArgs(pref).
			WillReturnError(errors.New("not found"))

		_, err := rep.Get(pref)
		assert.Error(t, err)
	})
}

func TestSet(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rep := New(db)
	url := model.New("pref1", "https://example.com", "user-1")

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO urls \\(prefix, original_url, user_id\\) VALUES \\(\\$1, \\$2, \\$3\\)").
			WithArgs(url.Prefix, url.OriginalURL, url.UserID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := rep.Set(url)
		assert.NoError(t, err)
	})

	t.Run("unique violation conflict", func(t *testing.T) {
		pgErr := &pgconn.PgError{
			Code:           pgerrcode.UniqueViolation,
			ConstraintName: "idx_urls_original_url_unique",
		}
		mock.ExpectExec("INSERT INTO urls \\(prefix, original_url, user_id\\) VALUES \\(\\$1, \\$2, \\$3\\)").
			WithArgs(url.Prefix, url.OriginalURL, url.UserID).
			WillReturnError(pgErr)

		rows := sqlmock.NewRows([]string{"id", "prefix", "original_url", "user_id", "is_deleted"}).
			AddRow("uuid-existing", "pref-existing", url.OriginalURL, url.UserID, false)

		mock.ExpectQuery("SELECT id, prefix, original_url, user_id, is_deleted FROM urls where original_url = \\$1").
			WithArgs(url.OriginalURL).
			WillReturnRows(rows)

		err := rep.Set(url)
		assert.Error(t, err)
		var conflictErr *errors2.OriginalURLConflictError
		assert.True(t, errors.As(err, &conflictErr))
	})

	t.Run("other exec error", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO urls \\(prefix, original_url, user_id\\) VALUES \\(\\$1, \\$2, \\$3\\)").
			WithArgs(url.Prefix, url.OriginalURL, url.UserID).
			WillReturnError(errors.New("db error"))

		err := rep.Set(url)
		assert.Error(t, err)
	})
}

func TestSetMany(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rep := New(db)
	urls := []model.URL{
		model.New("pref1", "https://example.com/1", "user-1"),
		model.New("pref2", "https://example.com/2", "user-1"),
	}

	t.Run("success", func(t *testing.T) {
		mock.ExpectPrepare("INSERT INTO urls \\(prefix, original_url, user_id\\) SELECT \\* FROM UNNEST\\(\\$1::text\\[\\]\\, \\$2::text\\[\\]\\, \\$3::uuid\\[\\]\\)").
			ExpectExec().
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(2, 2))

		err := rep.SetMany(urls)
		assert.NoError(t, err)
	})

	t.Run("prepare error", func(t *testing.T) {
		mock.ExpectPrepare("INSERT INTO urls").
			WillReturnError(errors.New("prepare error"))

		err := rep.SetMany(urls)
		assert.Error(t, err)
	})

	t.Run("exec error", func(t *testing.T) {
		mock.ExpectPrepare("INSERT INTO urls \\(prefix, original_url, user_id\\) SELECT \\* FROM UNNEST\\(\\$1::text\\[\\]\\, \\$2::text\\[\\]\\, \\$3::uuid\\[\\]\\)").
			ExpectExec().
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("exec error"))

		err := rep.SetMany(urls)
		assert.Error(t, err)
	})
}

func TestDeleteManyFromUserId(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rep := New(db)
	prefixes := []string{"pref1", "pref2"}
	userID := "user-1"

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec("UPDATE urls SET is_deleted = true WHERE user_id = \\$1 AND prefix IN \\(SELECT UNNEST\\(\\$2::text\\[\\]\\)\\)").
			WithArgs(userID, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 2))

		err := rep.DeleteManyFromUserId(prefixes, userID)
		assert.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		mock.ExpectExec("UPDATE urls SET is_deleted = true WHERE user_id = \\$1 AND prefix IN \\(SELECT UNNEST\\(\\$2::text\\[\\]\\)\\)").
			WithArgs(userID, sqlmock.AnyArg()).
			WillReturnError(errors.New("update error"))

		err := rep.DeleteManyFromUserId(prefixes, userID)
		assert.Error(t, err)
	})
}

func TestFindFromOriginal(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rep := New(db)
	orig := "https://example.com"

	t.Run("success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "prefix", "original_url", "user_id", "is_deleted"}).
			AddRow("uuid-1", "pref1", orig, "user-1", false)

		mock.ExpectQuery("SELECT id, prefix, original_url, user_id, is_deleted FROM urls where original_url = \\$1").
			WithArgs(orig).
			WillReturnRows(rows)

		url, err := rep.FindFromOriginal(orig)
		assert.NoError(t, err)
		assert.Equal(t, "pref1", url.Prefix)
	})
}

func TestFindFromUserID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rep := New(db)
	userID := "user-1"

	t.Run("success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "prefix", "original_url", "user_id", "is_deleted"}).
			AddRow("uuid-1", "pref1", "https://example.com/1", userID, false).
			AddRow("uuid-2", "pref2", "https://example.com/2", userID, false)

		mock.ExpectQuery("SELECT id, prefix, original_url, user_id, is_deleted FROM urls where user_id = \\$1").
			WithArgs(userID).
			WillReturnRows(rows)

		urls, err := rep.FindFromUserID(userID)
		assert.NoError(t, err)
		assert.Len(t, urls, 2)
	})

	t.Run("query error", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, prefix, original_url, user_id, is_deleted FROM urls where user_id = \\$1").
			WithArgs(userID).
			WillReturnError(errors.New("query error"))

		_, err := rep.FindFromUserID(userID)
		assert.Error(t, err)
	})

	t.Run("rows error", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "prefix", "original_url", "user_id", "is_deleted"}).
			AddRow("uuid-1", "pref1", "https://example.com/1", userID, false).
			RowError(0, errors.New("row error"))

		mock.ExpectQuery("SELECT id, prefix, original_url, user_id, is_deleted FROM urls where user_id = \\$1").
			WithArgs(userID).
			WillReturnRows(rows)

		_, err := rep.FindFromUserID(userID)
		assert.Error(t, err)
	})
}

func BenchmarkInDB_Set(b *testing.B) {
	db, mock, err := sqlmock.New()
	require.NoError(b, err)
	defer db.Close()

	rep := New(db)
	url := model.New("pref1", "https://example.com", "user-1")

	b.ResetTimer()
	for b.Loop() {
		mock.ExpectExec("INSERT INTO urls \\(prefix, original_url, user_id\\) VALUES \\(\\$1, \\$2, \\$3\\)").
			WithArgs(url.Prefix, url.OriginalURL, url.UserID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		_ = rep.Set(url)
	}
}

func BenchmarkInDB_Get(b *testing.B) {
	db, mock, err := sqlmock.New()
	require.NoError(b, err)
	defer db.Close()

	rep := New(db)
	uID := "uuid-1"
	pref := "pref1"
	orig := "https://example.com"
	userID := "user-1"
	isDeleted := false

	b.ResetTimer()
	for b.Loop() {
		rows := sqlmock.NewRows([]string{"id", "prefix", "original_url", "user_id", "is_deleted"}).
			AddRow(uID, pref, orig, userID, isDeleted)

		mock.ExpectQuery("SELECT id, prefix, original_url, user_id, is_deleted FROM urls where prefix = \\$1").
			WithArgs(pref).
			WillReturnRows(rows)

		_, _ = rep.Get(pref)
	}
}

func BenchmarkInDB_FindFromOriginal(b *testing.B) {
	db, mock, err := sqlmock.New()
	require.NoError(b, err)
	defer db.Close()

	rep := New(db)
	orig := "https://example.com"

	b.ResetTimer()
	for b.Loop() {
		rows := sqlmock.NewRows([]string{"id", "prefix", "original_url", "user_id", "is_deleted"}).
			AddRow("uuid-1", "pref1", orig, "user-1", false)

		mock.ExpectQuery("SELECT id, prefix, original_url, user_id, is_deleted FROM urls where original_url = \\$1").
			WithArgs(orig).
			WillReturnRows(rows)

		_, _ = rep.FindFromOriginal(orig)
	}
}

func BenchmarkInDB_FindFromUserID(b *testing.B) {
	db, mock, err := sqlmock.New()
	require.NoError(b, err)
	defer db.Close()

	rep := New(db)
	userID := "user-1"

	b.ResetTimer()
	for b.Loop() {
		rows := sqlmock.NewRows([]string{"id", "prefix", "original_url", "user_id", "is_deleted"}).
			AddRow("uuid-1", "pref1", "https://example.com/1", userID, false).
			AddRow("uuid-2", "pref2", "https://example.com/2", userID, false)

		mock.ExpectQuery("SELECT id, prefix, original_url, user_id, is_deleted FROM urls where user_id = \\$1").
			WithArgs(userID).
			WillReturnRows(rows)

		_, _ = rep.FindFromUserID(userID)
	}
}
