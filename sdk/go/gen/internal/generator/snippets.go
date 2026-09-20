package generator

// The Go fragments the generator emits, written as they appear in the generated
// file. Numbered verbs let one value repeat across a fragment, and gofmt
// formats the assembled source

const (
	structTypeOpen  = "type %s struct {\n"
	structTypeField = "%s %s\n"

	codecStructOpen = "%s.Struct(\n"

	codecFieldDecl = `%[1]s.Field(%[2]q, %[3]s,
func(v *%[4]s) *%[5]s {
return &v.%[6]s
},
),
`

	emptyCodecDecl = `%[1]s := %[2]s.Struct[%[3]s]()`

	codecDecl = `%[1]s := %[2]s`

	// a self-referential initializer is an initialization cycle
	lazyCodecDecl = `var %[1]sImpl %[2]s.%[3]s[%[4]s]

%[1]s := %[2]s.Ref(&%[1]sImpl)

%[1]sImpl = %[5]s`

	loggedHandler = `logged(%q, %s)`

	syncHandlerDecl = `%[1]s
%[2]s, %[3]s,
func(in %[4]s) (%[5]s, error) {
%[6]s
})`

	compHandlerDecl = `%[1]s
%[2]s,
func(in %[3]s) error {
%[4]s
})`

	wrapErrCheck = `if err != nil {
return %s{}, err
}
`
)
