package main

import (
	"bytes"
	"html/template"
	"io/ioutil"
	"sort"

	"github.com/StackExchange/dnscontrol/providers"
	_ "github.com/StackExchange/dnscontrol/providers/_all"
)

func generateFeatureMatrix() error {
	allNames := map[string]bool{}
	for n := range providers.RegistrarTypes {
		allNames[n] = true
	}
	for n := range providers.DNSProviderTypes {
		allNames[n] = true
	}
	providerTypes := []string{}
	for n := range allNames {
		providerTypes = append(providerTypes, n)
	}
	sort.Strings(providerTypes)
	matrix := &FeatureMatrix{
		Providers: map[string]FeatureMap{},
		Features: []FeatureDef{
			{"Official Support", "This means the provider is actively used at Stack Exchange, bugs are more likely to be fixed, and failing integration tests will block a release. See below for details"},
			{"Registrar", "The provider has registrar capabilities to set nameservers for zones"},
			{"DNS Provider", "Can manage and serve DNS zones"},
			{"ALIAS", "Provider supports some kind of ALIAS, ANAME or flattened CNAME record type"},
			{"SRV", "Driver has explicitly implemented SRV record management"},
			{"PTR", "Provider supports adding PTR records for reverse lookup zones"},
			{"CAA", "Provider can manage CAA records"},

			{"dual host", "This provider is recommended for use in 'dual hosting' scenarios. Usually this means the provider allows full control over the apex NS records"},
			{"create-domains", "This means the provider can automatically create domains that do not currently exist on your account. The 'dnscontrol create-domains' command will initialize any missing domains"},
			{"no_purge", "indicates you can use NO_PURGE macro to prevent deleting records not managed by dnscontrol. A few providers that generate the entire zone from scratch have a problem implementing this."},
		},
	}
	for _, p := range providerTypes {
		if p == "NONE" {
			continue
		}
		fm := FeatureMap{}
		notes := providers.Notes[p]
		if notes == nil {
			notes = providers.DocumentationNotes{}
		}
		setCap := func(name string, cap providers.Capability) {
			if notes[cap] != nil {
				fm[name] = notes[cap]
				return
			}
			fm.SetSimple(name, true, func() bool { return providers.ProviderHasCabability(p, cap) })
		}
		setDoc := func(name string, cap providers.Capability) {
			if notes[cap] != nil {
				fm[name] = notes[cap]
			}
		}
		setDoc("Official Support", providers.DocOfficiallySupported)
		fm.SetSimple("Registrar", false, func() bool { return providers.RegistrarTypes[p] != nil })
		fm.SetSimple("DNS Provider", false, func() bool { return providers.DNSProviderTypes[p] != nil })
		setCap("ALIAS", providers.CanUseAlias)
		setCap("SRV", providers.CanUseSRV)
		setCap("PTR", providers.CanUsePTR)
		setCap("CAA", providers.CanUseCAA)
		setDoc("dual host", providers.DocDualHost)
		setDoc("create-domains", providers.DocCreateDomains)

		// no purge is a freaky double negative
		cap := providers.CantUseNOPURGE
		if notes[cap] != nil {
			fm["no_purge"] = notes[cap]
		} else {
			fm.SetSimple("no_purge", false, func() bool { return !providers.ProviderHasCabability(p, cap) })
		}
		matrix.Providers[p] = fm
	}
	buf := &bytes.Buffer{}
	err := tmpl.Execute(buf, matrix)
	if err != nil {
		return err
	}
	return ioutil.WriteFile("docs/_includes/matrix.html", buf.Bytes(), 0644)
}

type FeatureDef struct {
	Name, Desc string
}
type FeatureMap map[string]*providers.DocumentationNote

func (fm FeatureMap) SetSimple(name string, unknownsAllowed bool, f func() bool) {
	if f() {
		fm[name] = &providers.DocumentationNote{HasFeature: true}
	} else if !unknownsAllowed {
		fm[name] = &providers.DocumentationNote{HasFeature: false}
	}
}

type FeatureMatrix struct {
	Features  []FeatureDef
	Providers map[string]FeatureMap
}

var tmpl = template.Must(template.New("").Funcs(template.FuncMap{
	"safe": func(s string) template.HTML { return template.HTML(s) },
}).Parse(`
	{% comment %}
    Matrix generated by build/generate/featureMatrix.go. DO NOT HAND EDIT! 
{% endcomment %}{{$providers := .Providers}}
<table class="table-header-rotated">
<thead>
	<tr>
	<th></th>
	{{range $key,$val := $providers}}<th class="rotate"><div><span>{{$key}}</span></div></th>
	{{end -}}
	</tr>
</thead>
<tbody>
	{{range .Features}}{{$name := .Name}}<tr>
		<th class="row-header" style="text-decoration: underline;" data-toggle="tooltip" data-container="body" data-placement="top" title="{{.Desc}}">{{$name}}</th>
		{{range $pname, $features := $providers}}{{$f := index $features $name}}{{if $f -}}
		<td class="{{if $f.HasFeature}}success{{else}}danger{{end}}"
			{{- if $f.Comment}} data-toggle="tooltip" data-container="body" data-placement="top" title="{{$f.Comment}}"{{end}}>
			<i class="fa {{if $f.Comment}}has-tooltip {{end}}
				{{- if $f.HasFeature}}fa-check text-success{{else}}fa-times text-danger{{end}}" aria-hidden="true"></i>
		</td>
		{{- else}}<td><i class="fa fa-minus dim"></i></td>{{end}}
		{{end -}}
	</tr>
	{{end -}}
</tbody>
</table>
`))
