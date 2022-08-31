/*
2021 © Postgres.ai
*/

// Package db provides database helpers.
package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v4"
	"golang.org/x/mod/semver"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

// ConnectionString builds PostgreSQL connection string.
func ConnectionString(host, port, username, dbname, password string) string {
	return fmt.Sprintf("host=%s port=%s user='%s' database='%s' password='%s'", host, port, username, dbname, password)
}

const (
	availableExtensions = "select name, default_version, coalesce(installed_version,'') from pg_available_extensions " +
		"where installed_version is not null"
	availableLocales = "select datname, lower(datcollate), lower(datctype) from pg_catalog.pg_database"
)

type extension struct {
	name             string
	defaultVersion   string
	installedVersion string
}

type locale struct {
	name    string
	collate string
	ctype   string
}

// CheckSource checks the readiness of the source database to dump and restore processes.
func CheckSource(ctx context.Context, conf *models.ConnectionTest, imageContent *ImageContent) (*models.TestConnection, error) {
	connStr := ConnectionString(conf.Host, conf.Port, conf.Username, conf.DBName, conf.Password)

	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		log.Dbg("failed to test database connection:", err)

		return &models.TestConnection{
			Status:  models.TCStatusError,
			Result:  models.TCResultConnectionError,
			Message: err.Error(),
		}, nil
	}

	defer func() {
		if err := conn.Close(ctx); err != nil {
			log.Dbg("failed to close connection:", err)
		}
	}()

	var one int

	if err := conn.QueryRow(ctx, "select 1").Scan(&one); err != nil {
		return &models.TestConnection{
			Status:  models.TCStatusError,
			Result:  models.TCResultConnectionError,
			Message: err.Error(),
		}, nil
	}

	if !imageContent.IsReady() {
		return &models.TestConnection{
			Status: models.TCStatusNotice,
			Result: models.TCResultUnexploredImage,
			Message: "The connection to the database was successful. " +
				"Details about the extensions and locales of the Docker image have not yet been collected. Please try again later",
		}, nil
	}

	if missing, unsupported, err := checkExtensions(ctx, conn, imageContent.Extensions()); err != nil {
		return &models.TestConnection{
			Status:  models.TCStatusWarning,
			Result:  models.TCResultMissingExtension,
			Message: buildExtensionsWarningMessage(missing, unsupported),
		}, nil
	}

	if missing, err := checkLocales(ctx, conn, imageContent.Locales(), imageContent.Databases()); err != nil {
		return &models.TestConnection{
			Status:  models.TCStatusWarning,
			Result:  models.TCResultMissingLocale,
			Message: buildLocalesWarningMessage(missing),
		}, nil
	}

	return &models.TestConnection{
		Status:  models.TCStatusOK,
		Result:  models.TCResultOK,
		Message: models.TCMessageOK,
	}, nil
}

func checkExtensions(ctx context.Context, conn *pgx.Conn, imageExtensions map[string]string) ([]extension, []extension, error) {
	rows, err := conn.Query(ctx, availableExtensions)
	if err != nil {
		return nil, nil, err
	}

	missingExtensions := []extension{}
	unsupportedVersions := []extension{}

	for rows.Next() {
		var ext extension
		if err := rows.Scan(&ext.name, &ext.defaultVersion, &ext.installedVersion); err != nil {
			return nil, nil, err
		}

		imageExt, ok := imageExtensions[ext.name]
		if !ok {
			missingExtensions = append(missingExtensions, ext)
			continue
		}

		if !semver.IsValid(toCanonicalSemver(ext.defaultVersion)) {
			unsupportedVersions = append(unsupportedVersions, ext)
			continue
		}

		if semver.Compare(toCanonicalSemver(imageExt), toCanonicalSemver(ext.defaultVersion)) == -1 {
			unsupportedVersions = append(unsupportedVersions, ext)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	if len(missingExtensions) != 0 || len(unsupportedVersions) != 0 {
		return missingExtensions, unsupportedVersions, errors.New("extension warning found")
	}

	return nil, nil, nil
}

func toCanonicalSemver(v string) string {
	if v == "" {
		return ""
	}

	if v[0] != 'v' {
		return "v" + v
	}

	return v
}

func buildExtensionsWarningMessage(missingExtensions, unsupportedVersions []extension) string {
	sb := &strings.Builder{}

	if len(missingExtensions) > 0 {
		sb.WriteString("There are missing extensions:")

		formatExtensionList(sb, missingExtensions)

		sb.WriteRune('\n')
	}

	if len(unsupportedVersions) > 0 {
		sb.WriteString("There are extensions with an unsupported version:")

		formatExtensionList(sb, unsupportedVersions)
	}

	return sb.String()
}

func formatExtensionList(sb *strings.Builder, extensions []extension) {
	length := len(extensions)

	for i, missing := range extensions {
		sb.WriteString(" " + missing.name + " " + missing.defaultVersion)

		if i != length-1 {
			sb.WriteRune(',')
		}
	}
}

func checkLocales(ctx context.Context, conn *pgx.Conn, imageLocales, databases map[string]struct{}) ([]locale, error) {
	rows, err := conn.Query(ctx, availableLocales)
	if err != nil {
		return nil, err
	}

	missingLocales := []locale{}

	for rows.Next() {
		var l locale
		if err := rows.Scan(&l.name, &l.collate, &l.ctype); err != nil {
			return nil, err
		}

		if _, ok := databases[l.name]; len(databases) > 0 && !ok {
			// Skip the check if there is a list of restored databases, and it does not contain the current database.
			continue
		}

		cleanCollate := strings.ReplaceAll(strings.ToLower(l.collate), "-", "")

		if _, ok := imageLocales[cleanCollate]; !ok {
			missingLocales = append(missingLocales, l)
			continue
		}

		cleanCtype := strings.ReplaceAll(strings.ToLower(l.ctype), "-", "")

		if _, ok := imageLocales[cleanCtype]; !ok {
			missingLocales = append(missingLocales, l)
			continue
		}
	}

	if len(missingLocales) != 0 {
		return missingLocales, errors.New("locale warning found")
	}

	return nil, nil
}

func buildLocalesWarningMessage(missingLocales []locale) string {
	sb := &strings.Builder{}

	if length := len(missingLocales); length > 0 {
		sb.WriteString("There are missing locales:")

		for i, missing := range missingLocales {
			sb.WriteString(fmt.Sprintf(" '%s' (collate: %s, ctype: %s)", missing.name, missing.collate, missing.ctype))

			if i != length-1 {
				sb.WriteRune(',')
			}
		}
	}

	return sb.String()
}
