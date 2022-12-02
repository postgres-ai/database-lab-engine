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

const (
	availableExtensions = "select name, default_version, coalesce(installed_version,'') from pg_available_extensions " +
		"where installed_version is not null"

	availableLocales = "select datname, lower(datcollate), lower(datctype) from pg_catalog.pg_database"

	availableDBsTemplate = `select datname from pg_catalog.pg_database 
	where not datistemplate and has_database_privilege('%s', datname, 'CONNECT')`

	// maxNumberVerifiedDBs defines the maximum number of databases to verify availability as a database source.
	// The DB source instance can contain a large number of databases, so the verification will take a long time.
	// Therefore, we introduced a limit on the maximum number of databases to check for suitability as a source.
	maxNumberVerifiedDBs = 5
)

var (
	errExtensionWarning = errors.New("extension warning found")
	errLocaleWarning    = errors.New("locale warning found")
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

// ConnectionString builds PostgreSQL connection string.
func ConnectionString(host, port, username, dbname, password string) string {
	return fmt.Sprintf("host=%s port=%s user='%s' database='%s' password='%s'", host, port, username, dbname, password)
}

// GetDatabaseListQuery provides the query to get the list of databases available for user.
func GetDatabaseListQuery(username string) string {
	return fmt.Sprintf(availableDBsTemplate, username)
}

// CheckSource checks the readiness of the source database to dump and restore processes.
func CheckSource(ctx context.Context, conf *models.ConnectionTest, imageContent *ImageContent) (*models.TestConnection, error) {
	conn, tcResponse := checkConnection(ctx, conf, conf.DBName)
	if tcResponse != nil {
		return tcResponse, nil
	}

	defer func() {
		if err := conn.Close(ctx); err != nil {
			log.Dbg("failed to close connection:", err)
		}
	}()

	dbList := conf.DBList

	if len(dbList) == 0 {
		dbSourceList, err := getDBList(ctx, conn, conf.Username)
		if err != nil {
			return nil, err
		}

		dbList = dbSourceList
	}

	if len(dbList) > maxNumberVerifiedDBs {
		dbList = dbList[:maxNumberVerifiedDBs]
		tcResponse = &models.TestConnection{
			Status: models.TCStatusNotice,
			Result: models.TCResultUnverifiedDB,
			Message: "Too many databases were requested to be checked. Only the following databases have been verified: " +
				strings.Join(dbList, ", "),
		}
	}

	for _, dbName := range dbList {
		dbConn, listTC := checkConnection(ctx, conf, dbName)
		if listTC != nil {
			return listTC, nil
		}

		listTC, err := checkContent(ctx, dbConn, dbName, imageContent)
		if err != nil {
			return nil, err
		}

		if listTC != nil {
			return listTC, nil
		}
	}

	if tcResponse != nil {
		return tcResponse, nil
	}

	return &models.TestConnection{
		Status:  models.TCStatusOK,
		Result:  models.TCResultOK,
		Message: models.TCMessageOK,
	}, nil
}

func getDBList(ctx context.Context, conn *pgx.Conn, dbUsername string) ([]string, error) {
	dbList := make([]string, 0)

	rows, err := conn.Query(ctx, GetDatabaseListQuery(dbUsername))
	if err != nil {
		return nil, fmt.Errorf("failed to perform query listing databases: %w", err)
	}

	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			return nil, fmt.Errorf("failed to scan next row in database list result set: %w", err)
		}

		dbList = append(dbList, dbName)
	}

	return dbList, nil
}

func checkConnection(ctx context.Context, conf *models.ConnectionTest, dbName string) (*pgx.Conn, *models.TestConnection) {
	connStr := ConnectionString(conf.Host, conf.Port, conf.Username, dbName, conf.Password)

	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		log.Dbg("failed to test database connection:", err)

		return nil, &models.TestConnection{
			Status:  models.TCStatusError,
			Result:  models.TCResultConnectionError,
			Message: err.Error(),
		}
	}

	var one int

	if err := conn.QueryRow(ctx, "select 1").Scan(&one); err != nil {
		return nil, &models.TestConnection{
			Status:  models.TCStatusError,
			Result:  models.TCResultConnectionError,
			Message: err.Error(),
		}
	}

	return conn, nil
}

func checkContent(ctx context.Context, conn *pgx.Conn, dbName string, imageContent *ImageContent) (*models.TestConnection, error) {
	if !imageContent.IsReady() {
		return &models.TestConnection{
			Status: models.TCStatusNotice,
			Result: models.TCResultUnexploredImage,
			Message: "The connection to the database was successful. " +
				"Details about the extensions and locales of the Docker image have not yet been collected. Please try again later",
		}, nil
	}

	if missing, unsupported, err := checkExtensions(ctx, conn, imageContent.Extensions()); err != nil {
		if err != errExtensionWarning {
			return nil, fmt.Errorf("failed to check database extensions: %w", err)
		}

		return &models.TestConnection{
			Status:  models.TCStatusWarning,
			Result:  models.TCResultMissingExtension,
			Message: buildExtensionsWarningMessage(dbName, missing, unsupported),
		}, nil
	}

	if missing, err := checkLocales(ctx, conn, imageContent.Locales(), imageContent.Databases()); err != nil {
		if err != errLocaleWarning {
			return nil, fmt.Errorf("failed to check database locales: %w", err)
		}

		return &models.TestConnection{
			Status:  models.TCStatusWarning,
			Result:  models.TCResultMissingLocale,
			Message: buildLocalesWarningMessage(dbName, missing),
		}, nil
	}

	return nil, nil
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
		return missingExtensions, unsupportedVersions, errExtensionWarning
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

func buildExtensionsWarningMessage(dbName string, missingExtensions, unsupportedVersions []extension) string {
	sb := &strings.Builder{}

	if len(missingExtensions) > 0 {
		sb.WriteString("The image specified in section \"databaseContainer\" lacks the following " +
			"extensions used in the source database (\"" + dbName + "\"):")

		formatExtensionList(sb, missingExtensions)

		sb.WriteString(".\n")
	}

	if len(unsupportedVersions) > 0 {
		sb.WriteString("The source database (\"" + dbName + "\") uses extensions that are present " +
			"in image specified in section \"databaseContainer\" but their versions are not supported by the image:")

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
		return missingLocales, errLocaleWarning
	}

	return nil, nil
}

func buildLocalesWarningMessage(dbName string, missingLocales []locale) string {
	sb := &strings.Builder{}

	if length := len(missingLocales); length > 0 {
		sb.WriteString("The image specified in section \"databaseContainer\" lacks the following " +
			"locales used in the source database (\"" + dbName + "\"):")

		for i, missing := range missingLocales {
			sb.WriteString(fmt.Sprintf(" '%s' (collate: %s, ctype: %s)", missing.name, missing.collate, missing.ctype))

			if i != length-1 {
				sb.WriteRune(',')
			}
		}
	}

	return sb.String()
}
