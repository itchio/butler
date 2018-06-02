package hades

type AssocMode int

const (
	AssocModeAppend AssocMode = iota
	AssocModeReplace
)

type assocField struct {
	name     string
	search   *SearchParams
	mode     AssocMode
	children []AssocField
}

type saveParams struct {
	assocs   []AssocField
	omitRoot bool
}

type preloadParams struct {
	assocs []AssocField
}

type SaveParam interface {
	ApplyToSaveParams(sp *saveParams)
}

type PreloadParam interface {
	ApplyToPreloadParams(pp *preloadParams)
}

type AssocField interface {
	SaveParam
	PreloadParam
	Name() string
	Mode() AssocMode
	Search() *SearchParams
	Children() []AssocField
}

// -------------

// OmitRoot tells save to not save the record passed,
// but only associations
func OmitRoot() SaveParam {
	return &omitRoot{}
}

type omitRoot struct{}

func (o *omitRoot) ApplyToSaveParams(sp *saveParams) {
	sp.omitRoot = true
}

// Assoc tells save to save the specified association,
// but not to remove any existing associated records, even if
// they're not listed anymore
func Assoc(fieldName string, children ...AssocField) AssocField {
	return &assocField{
		name:     fieldName,
		mode:     AssocModeAppend,
		children: children,
	}
}

// AssocReplace tells save to save the specified assocation,
// and to remove any associated records that are no longer listed
func AssocReplace(fieldName string, children ...AssocField) AssocField {
	return &assocField{
		name:     fieldName,
		mode:     AssocModeReplace,
		children: children,
	}
}

func AssocWithSearch(fieldName string, search *SearchParams, children ...AssocField) AssocField {
	return &assocField{
		name:     fieldName,
		mode:     AssocModeAppend,
		search:   search,
		children: children,
	}
}

func (f *assocField) ApplyToSaveParams(sp *saveParams) {
	sp.assocs = append(sp.assocs, f)
}

func (f *assocField) ApplyToPreloadParams(pp *preloadParams) {
	pp.assocs = append(pp.assocs, f)
}

func (f *assocField) Name() string {
	return f.name
}

func (f *assocField) Mode() AssocMode {
	return f.mode
}

func (f *assocField) Children() []AssocField {
	return f.children
}

func (f *assocField) Search() *SearchParams {
	return f.search
}
