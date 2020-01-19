package postman

import "time"

type Environment struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Values               []Values  `json:"values"`
	PostmanVariableScope string    `json:"_postman_variable_scope"`
	PostmanExportedAt    time.Time `json:"_postman_exported_at"`
	PostmanExportedUsing string    `json:"_postman_exported_using"`
}

type Values struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

// Map maps the postman environment to golang map[string]string
func (env *Environment) Map() map[string]string {
	out := map[string]string{}
	for _, v := range env.Values {
		out[v.Key] = v.Value
	}
	return out
}
