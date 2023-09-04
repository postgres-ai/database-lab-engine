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

	dbVersionQuery = `select setting::integer/10000 from pg_settings where name = 'server_version_num'`

	tuningParamsQuery = `select 
  name, setting
from
  pg_settings
where
  source <> 'default'
  and (
    name ~ '(work_mem$|^enable_|_cost$|scan_size$|effective_cache_size|^jit)'
    or name ~ '(^geqo|default_statistics_target|constraint_exclusion|cursor_tuple_fraction)'
    or name ~ '(collapse_limit$|parallel|plan_cache_mode)'
  )`

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

type tuningParam struct {
	name    string
	setting string
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
func CheckSource(ctx context.Context, conf *models.ConnectionTest, imageContent *ImageContent) (*models.DBSource, error) {
	dbSource := &models.DBSource{}

	conn, tcResponse := checkConnection(ctx, conf, conf.DBName)
	if tcResponse != nil {
		dbSource.TestConnection = tcResponse
		return dbSource, nil
	}

	defer func() {
		if err := conn.Close(ctx); err != nil {
			log.Dbg("failed to close connection:", err)
		}
	}()

	// Return the database version in any case.
	dbVersion, err := getMajorVersion(ctx, conn)
	if err != nil {
		return nil, err
	}

	dbSource.DBVersion = dbVersion

	tcResponse = &models.TestConnection{
		Status:  models.TCStatusOK,
		Result:  models.TCResultOK,
		Message: models.TCMessageOK,
	}

	dbSource.TestConnection = tcResponse

	tuningParameters, err := getTuningParameters(ctx, conn)
	if err != nil {
		dbSource.Status = models.TCStatusError
		dbSource.Result = models.TCResultQueryError
		dbSource.Message = err.Error()

		return dbSource, err
	}

	dbSource.TuningParams = tuningParameters

	dbCheck, err := checkDatabases(ctx, conn, conf, imageContent)
	if err != nil {
		dbSource.Status = models.TCStatusError
		dbSource.Result = models.TCResultQueryError
		dbSource.Message = err.Error()

		return dbSource, err
	}

	dbSource.TestConnection = dbCheck

	return dbSource, nil
}

func checkDatabases(ctx context.Context, conn *pgx.Conn, conf *models.ConnectionTest,
	imageContent *ImageContent) (*models.TestConnection, error) {
	var tcResponse = &models.TestConnection{
		Status:  models.TCStatusOK,
		Result:  models.TCResultOK,
		Message: models.TCMessageOK,
	}

	dbList := conf.DBList

	if len(dbList) == 0 {
		dbSourceList, err := getDBList(ctx, conn, conf.Username)
		if err != nil {
			return nil, err
		}

		dbList = dbSourceList
	}

	dbReport := make(map[string]*checkContentResp, 0)

	if len(dbList) > maxNumberVerifiedDBs {
		dbList = dbList[:maxNumberVerifiedDBs]
		dbReport[""] = &checkContentResp{
			status: models.TCStatusNotice,
			result: models.TCResultUnverifiedDB,
			message: "Too many databases. Only checked these databases: " +
				strings.Join(dbList, ", ") + ". ",
		}
	}

	for _, dbName := range dbList {
		dbConn, listTC := checkConnection(ctx, conf, dbName)
		if listTC != nil {
			dbReport[dbName] = &checkContentResp{
				status:  listTC.Status,
				result:  listTC.Result,
				message: listTC.Message,
			}

			continue
		}

		contentChecks := checkDBContent(ctx, dbConn, imageContent)

		if contentChecks != nil {
			dbReport[dbName] = contentChecks
		}
	}

	if len(dbReport) > 0 {
		tcResponse = aggregate(tcResponse, dbReport)
	}

	return tcResponse, nil
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

type aggregateState struct {
	general        string
	errors         map[string]string
	missingExt     map[string][]extension
	unsupportedExt map[string][]extension
	missingLocales map[string][]locale
	unexploredDBs  []string
}

func newAggregateState() aggregateState {
	return aggregateState{
		general:        "",
		errors:         make(map[string]string, 0),
		missingExt:     make(map[string][]extension, 0),
		unsupportedExt: make(map[string][]extension, 0),
		missingLocales: make(map[string][]locale, 0),
		unexploredDBs:  make([]string, 0),
	}
}

func aggregate(tcResponse *models.TestConnection, collection map[string]*checkContentResp) *models.TestConnection {
	agg := newAggregateState()
	sb := strings.Builder{}

	for dbName, contentResponse := range collection {
		if contentResponse.status > tcResponse.Status {
			tcResponse.Status = contentResponse.status
			tcResponse.Result = contentResponse.result
		}

		switch contentResponse.result {
		case models.TCResultUnverifiedDB:
			agg.general += contentResponse.message

		case models.TCResultQueryError, models.TCResultConnectionError:
			agg.errors[dbName] = contentResponse.message

		case models.TCResultMissingExtension:
			if len(contentResponse.missingExt) > 0 {
				agg.missingExt[dbName] = append(agg.missingExt[dbName], contentResponse.missingExt...)
			}

			if len(contentResponse.unsupportedExt) > 0 {
				agg.unsupportedExt[dbName] = append(agg.unsupportedExt[dbName], contentResponse.unsupportedExt...)
			}

		case models.TCResultMissingLocale:
			agg.missingLocales[dbName] = append(agg.missingLocales[dbName], contentResponse.missingLocales...)

		case models.TCResultUnexploredImage:
			agg.unexploredDBs = append(agg.unexploredDBs, dbName)

		case models.TCResultOK:
		default:
		}
	}

	sb.WriteString(agg.general)
	sb.WriteString(buildErrorMessage(agg.errors))
	sb.WriteString(buildExtensionsWarningMessage(agg.missingExt, agg.unsupportedExt))
	sb.WriteString(buildLocalesWarningMessage(agg.missingLocales))
	sb.WriteString(unexploredDBsNoticeMessage(agg.unexploredDBs))

	tcResponse.Message = sb.String()

	return tcResponse
}

func buildErrorMessage(errors map[string]string) string {
	if len(errors) == 0 {
		return ""
	}

	sb := strings.Builder{}
	sb.WriteString("Issues detected in databases:\n")

	for dbName, message := range errors {
		sb.WriteString(fmt.Sprintf(" %q - %s;\n", dbName, message))
	}

	sb.WriteString(" \n")

	return sb.String()
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

type checkContentResp struct {
	status         models.StatusType
	result         string
	message        string
	missingExt     []extension
	unsupportedExt []extension
	missingLocales []locale
}

func checkDBContent(ctx context.Context, conn *pgx.Conn, imageContent *ImageContent) *checkContentResp {
	if !imageContent.IsReady() {
		return &checkContentResp{
			status: models.TCStatusNotice,
			result: models.TCResultUnexploredImage,
			message: "Connected to database. " +
				"Docker image extensions and locales not yet analyzed. Retry later. ",
		}
	}

	if missing, unsupported, err := checkExtensions(ctx, conn, imageContent.Extensions()); err != nil {
		if err != errExtensionWarning {
			return &checkContentResp{
				status:  models.TCStatusError,
				result:  models.TCResultQueryError,
				message: fmt.Sprintf("failed to check database extensions: %s", err),
			}
		}

		return &checkContentResp{
			status:         models.TCStatusWarning,
			result:         models.TCResultMissingExtension,
			missingExt:     missing,
			unsupportedExt: unsupported,
		}
	}

	if missing, err := checkLocales(ctx, conn, imageContent.Locales(), imageContent.Databases()); err != nil {
		if err != errLocaleWarning {
			return &checkContentResp{
				status:  models.TCStatusError,
				result:  models.TCResultQueryError,
				message: fmt.Sprintf("failed to check database locales: %s", err),
			}
		}

		return &checkContentResp{
			status:         models.TCStatusWarning,
			result:         models.TCResultMissingLocale,
			missingLocales: missing,
		}
	}

	return nil
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

func buildExtensionsWarningMessage(missingExtensions, unsupportedVersions map[string][]extension) string {
	sb := &strings.Builder{}

	if len(missingExtensions) > 0 {
		sb.WriteString("Image configured in \"databaseContainer\" missing " +
			"extensions installed in source databases: ")

		formatExtensionList(sb, missingExtensions)
	}

	if len(unsupportedVersions) > 0 {
		sb.WriteString("Source databases have extensions with different versions " +
			"than image configured in \"databaseContainer\":")

		formatExtensionList(sb, unsupportedVersions)
	}

	return sb.String()
}

func formatExtensionList(sb *strings.Builder, extensionMap map[string][]extension) {
	var j int

	lengthDBs := len(extensionMap)

	for dbName, extensions := range extensionMap {
		lengthExt := len(extensions)

		sb.WriteString(" " + dbName + " (")

		for i, missing := range extensions {
			sb.WriteString(missing.name + " " + missing.defaultVersion)

			if i != lengthExt-1 {
				sb.WriteString(", ")
			}
		}

		sb.WriteString(")")

		if j != lengthDBs-1 {
			sb.WriteRune(';')
		}

		j++
	}

	sb.WriteString(". \n")
}

func unexploredDBsNoticeMessage(dbs []string) string {
	if len(dbs) == 0 {
		return ""
	}

	return fmt.Sprintf("Connected to databases: %s. "+
		"Docker image extensions and locales not analyzed. Retry later.\n",
		strings.Join(dbs, ","))
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

func buildLocalesWarningMessage(localeMap map[string][]locale) string {
	var j int

	sb := &strings.Builder{}

	if lengthDBs := len(localeMap); lengthDBs > 0 {
		sb.WriteString("Image configured in \"databaseContainer\" missing " +
			"locales from source databases: ")

		for dbName, missingLocales := range localeMap {
			lengthLoc := len(missingLocales)

			sb.WriteString(" " + dbName + " (")

			for i, missing := range missingLocales {
				sb.WriteString(fmt.Sprintf(" '%s' (collate: %s, ctype: %s)", missing.name, missing.collate, missing.ctype))

				if i != lengthLoc-1 {
					sb.WriteRune(',')
				}
			}

			sb.WriteString(")")

			if j != lengthDBs-1 {
				sb.WriteRune(';')
			}

			j++
		}

		sb.WriteString(". \n")
	}

	return sb.String()
}

func getMajorVersion(ctx context.Context, conn *pgx.Conn) (int, error) {
	var majorVersion int

	row := conn.QueryRow(ctx, dbVersionQuery)

	if err := row.Scan(&majorVersion); err != nil {
		return 0, fmt.Errorf("failed to perform query detecting major version: %w", err)
	}

	return majorVersion, nil
}

func getTuningParameters(ctx context.Context, conn *pgx.Conn) (map[string]string, error) {
	rows, err := conn.Query(ctx, tuningParamsQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to perform query detecting query tuning params: %w", err)
	}

	var tuningParams = make(map[string]string)

	for rows.Next() {
		var param tuningParam

		if err := rows.Scan(&param.name, &param.setting); err != nil {
			return nil, fmt.Errorf("failed to scan query tuning params: %w", err)
		}

		tuningParams[param.name] = param.setting
	}

	return tuningParams, nil
}
