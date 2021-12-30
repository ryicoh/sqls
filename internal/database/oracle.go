package database

import (
	"context"
	"database/sql"
	"strconv"

	_ "github.com/godror/godror"
	"github.com/lighttiger2505/sqls/dialect"
)

func init() {
	RegisterOpen("oracle", oracleOpen)
	RegisterFactory("oracle", NewOracleDBRepository)
}

func oracleOpen(dbConnCfg *DBConfig) (*DBConnection, error) {
	var (
		conn *sql.DB
	)
	DSName, err := genOracleConfig(dbConnCfg)
	if err != nil {
		return nil, err
	}

	conn, err = sql.Open("godror", DSName)
	if err != nil {
		return nil, err
	}

	conn.SetMaxIdleConns(DefaultMaxIdleConns)
	conn.SetMaxOpenConns(DefaultMaxOpenConns)

	return &DBConnection{
		Conn: conn,
	}, nil
}

func genOracleConfig(connCfg *DBConfig) (string, error) {
	if connCfg.DataSourceName != "" {
		return connCfg.DataSourceName, nil
	}

	host, port := connCfg.Host, connCfg.Port
	if host == "" {
		host = "127.0.0.1"
	}
	if port == 0 {
		port = 1521
	}
	DSName := connCfg.User + "/" + connCfg.Passwd + "@" + host + ":" + strconv.Itoa(port) + "/" + connCfg.DBName
	return DSName, nil
}

type OracleDBRepository struct {
	Conn *sql.DB
}

func NewOracleDBRepository(conn *sql.DB) DBRepository {
	return &OracleDBRepository{Conn: conn}
}

func (db *OracleDBRepository) Driver() dialect.DatabaseDriver {
	return dialect.DatabaseDriverOracle
}

func (db *OracleDBRepository) CurrentDatabase(ctx context.Context) (string, error) {
	row := db.Conn.QueryRowContext(ctx, "SELECT SYS_CONTEXT('USERENV','CURRENT_SCHEMA') FROM DUAL")
	var database string
	if err := row.Scan(&database); err != nil {
		return "", err
	}
	return database, nil
}

func (db *OracleDBRepository) Databases(ctx context.Context) ([]string, error) {
	// one DB per connection for Oracle
	rows, err := db.Conn.QueryContext(ctx, "SELECT USERNAME FROM SYS.ALL_USERS ORDER BY USERNAME")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	databases := []string{}
	for rows.Next() {
		var database string
		if err := rows.Scan(&database); err != nil {
			return nil, err
		}
		databases = append(databases, database)
	}
	return databases, nil
}

func (db *OracleDBRepository) CurrentSchema(ctx context.Context) (string, error) {
	return db.CurrentDatabase(ctx)
}

func (db *OracleDBRepository) Schemas(ctx context.Context) ([]string, error) {
	return db.Databases(ctx)
}

func (db *OracleDBRepository) SchemaTables(ctx context.Context) (map[string][]string, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
	SELECT OWNER, TABLE_NAME
      FROM SYS.ALL_TABLES 
  ORDER BY OWNER, TABLE_NAME
		`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	databaseTables := map[string][]string{}
	for rows.Next() {
		var schema, table string
		if err := rows.Scan(&schema, &table); err != nil {
			return nil, err
		}

		if arr, ok := databaseTables[schema]; ok {
			databaseTables[schema] = append(arr, table)
		} else {
			databaseTables[schema] = []string{table}
		}
	}
	return databaseTables, nil
}

func (db *OracleDBRepository) Tables(ctx context.Context) ([]string, error) {
	rows, err := db.Conn.QueryContext(ctx, "SELECT TABLE_NAME FROM USER_TABLES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tables := []string{}
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	return tables, nil
}

func (db *OracleDBRepository) DescribeDatabaseTable(ctx context.Context) ([]*ColumnDesc, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
SELECT
OWNER,
TABLE_NAME,
COLUMN_NAME,
DATA_TYPE,
NULLABLE,
'',
DATA_DEFAULT,
''
FROM SYS.ALL_TAB_COLUMNS
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tableInfos := []*ColumnDesc{}
	for rows.Next() {
		var tableInfo ColumnDesc
		err := rows.Scan(
			&tableInfo.Schema,
			&tableInfo.Table,
			&tableInfo.Name,
			&tableInfo.Type,
			&tableInfo.Null,
			&tableInfo.Key,
			&tableInfo.Default,
			&tableInfo.Extra,
		)
		if err != nil {
			return nil, err
		}
		tableInfos = append(tableInfos, &tableInfo)
	}
	return tableInfos, nil
}

func (db *OracleDBRepository) DescribeDatabaseTableBySchema(ctx context.Context, schemaName string) ([]*ColumnDesc, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
		SELECT
		OWNER,
		TABLE_NAME,
		COLUMN_NAME,
		DATA_TYPE,
		NULLABLE,
		'',
		DATA_DEFAULT,
		''
		FROM SYS.ALL_TAB_COLUMNS
		WHERE OWNER= ?
`, schemaName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tableInfos := []*ColumnDesc{}
	for rows.Next() {
		var tableInfo ColumnDesc
		err := rows.Scan(
			&tableInfo.Schema,
			&tableInfo.Table,
			&tableInfo.Name,
			&tableInfo.Type,
			&tableInfo.Null,
			&tableInfo.Key,
			&tableInfo.Default,
			&tableInfo.Extra,
		)
		if err != nil {
			return nil, err
		}
		tableInfos = append(tableInfos, &tableInfo)
	}
	return tableInfos, nil
}

func (db *OracleDBRepository) Exec(ctx context.Context, query string) (sql.Result, error) {
	return db.Conn.ExecContext(ctx, query)
}

func (db *OracleDBRepository) Query(ctx context.Context, query string) (*sql.Rows, error) {
	return db.Conn.QueryContext(ctx, query)
}
