package postman

// Environment represents a postman environment
type Environment struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Values []*Value `json:"values"`
}

// Value represents a postman environment value
type Value struct {
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
