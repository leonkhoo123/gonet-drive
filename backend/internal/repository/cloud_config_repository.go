package repository

import (
	"database/sql"
	"go-file-server/internal/model"
)

type CloudConfigRepository interface {
	ListEnabled() ([]model.CloudConfig, error)
	ListAllNotDeleted() ([]model.CloudConfig, error)
	Update(id int, value *string, isEnabled *bool, isDeleted *bool) (int64, error)
}

type SQLiteCloudConfigRepo struct {
	DB *sql.DB
}

func NewSQLiteCloudConfigRepo(db *sql.DB) *SQLiteCloudConfigRepo {
	return &SQLiteCloudConfigRepo{DB: db}
}

func (r *SQLiteCloudConfigRepo) ListEnabled() ([]model.CloudConfig, error) {
	query := `SELECT config_name, config_type, config_unit, config_value FROM cloud_config WHERE is_enabled = 1 AND is_deleted = 0`
	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []model.CloudConfig
	for rows.Next() {
		var c model.CloudConfig
		var unit, value sql.NullString
		if err := rows.Scan(&c.ConfigName, &c.ConfigType, &unit, &value); err != nil {
			return nil, err
		}
		if unit.Valid {
			c.ConfigUnit = &unit.String
		}
		if value.Valid {
			c.ConfigValue = &value.String
		}
		c.IsEnabled = true
		configs = append(configs, c)
	}
	return configs, nil
}

func (r *SQLiteCloudConfigRepo) ListAllNotDeleted() ([]model.CloudConfig, error) {
	query := `SELECT id, config_name, config_type, config_unit, config_value, is_enabled 
			  FROM cloud_config WHERE is_deleted = 0`
	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []model.CloudConfig
	for rows.Next() {
		var c model.CloudConfig
		var unit, value sql.NullString
		if err := rows.Scan(&c.ID, &c.ConfigName, &c.ConfigType, &unit, &value, &c.IsEnabled); err != nil {
			return nil, err
		}
		if unit.Valid {
			c.ConfigUnit = &unit.String
		}
		if value.Valid {
			c.ConfigValue = &value.String
		}
		configs = append(configs, c)
	}
	return configs, nil
}

func (r *SQLiteCloudConfigRepo) Update(id int, value *string, isEnabled *bool, isDeleted *bool) (int64, error) {
	updates := []string{}
	args := []interface{}{}

	if value != nil {
		updates = append(updates, "config_value = ?")
		args = append(args, *value)
	}
	if isEnabled != nil {
		updates = append(updates, "is_enabled = ?")
		args = append(args, *isEnabled)
	}
	if isDeleted != nil {
		updates = append(updates, "is_deleted = ?")
		args = append(args, *isDeleted)
	}

	if len(updates) == 0 {
		return 0, nil
	}

	query := "UPDATE cloud_config SET "
	for i, u := range updates {
		query += u
		if i < len(updates)-1 {
			query += ", "
		}
	}
	query += " WHERE id = ? AND is_deleted = 0"
	args = append(args, id)

	result, err := r.DB.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
