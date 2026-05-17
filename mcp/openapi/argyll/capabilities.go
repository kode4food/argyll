package argyll

type (
	Capabilities struct {
		AttributeMapping MappingCapability  `json:"attribute_mapping"`
		EndpointArgs     EndpointCapability `json:"endpoint_args"`
		RequiredMatch    MatchCapability    `json:"required_match"`
	}

	MappingCapability struct {
		InputLocations  []string `json:"input_locations"`
		OutputLocation  string   `json:"output_location"`
		RenameField     string   `json:"rename_field"`
		TransformField  string   `json:"transform_field"`
		ScriptLanguages []string `json:"script_languages"`
		Supported       bool     `json:"supported"`
	}

	MatchCapability struct {
		Location        string   `json:"location"`
		ScriptLanguages []string `json:"script_languages"`
		Supported       bool     `json:"supported"`
	}

	EndpointCapability struct {
		PlaceholderSyntax string   `json:"placeholder_syntax"`
		Locations         []string `json:"locations"`
		Supported         bool     `json:"supported"`
	}
)

func Defaults() Capabilities {
	return Capabilities{
		AttributeMapping: MappingCapability{
			Supported:       true,
			InputLocations:  []string{"required.mapping", "optional.mapping"},
			OutputLocation:  "output.mapping",
			RenameField:     "name",
			TransformField:  "script",
			ScriptLanguages: []string{"jpath", "ale", "lua"},
		},
		RequiredMatch: MatchCapability{
			Supported:       true,
			Location:        "required.match",
			ScriptLanguages: []string{"ale", "lua", "jpath"},
		},
		EndpointArgs: EndpointCapability{
			Supported:         true,
			Locations:         []string{"path", "query"},
			PlaceholderSyntax: "{attribute_name}",
		},
	}
}
