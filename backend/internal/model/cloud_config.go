package model

type CloudConfig struct {
	ID          int     `json:"id"`
	ConfigName  string  `json:"config_name"`
	ConfigType  string  `json:"config_type"`
	ConfigUnit  *string `json:"config_unit"`
	ConfigValue *string `json:"config_value"`
	IsEnabled   bool    `json:"is_enabled"`
	IsDeleted   bool    `json:"is_deleted"`
}
