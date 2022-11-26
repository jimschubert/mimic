{{define "PullTemplate"}} ({{if .IsPull -}}
{{if .PullURL}}[contributed]({{.PullURL}}){{else}}contributed{{end}} by {{end}}{{if .AuthorURL -}}
[{{.Author}}]({{.AuthorURL}}){{else}}{{.Author}}{{end -}})
{{- end -}}
{{define "CommitTemplate" -}}
{{if .CommitURL}}[{{.CommitHashShort}}]({{.CommitURL}}){{else}}{{.CommitHashShort}}{{end -}}
{{- end -}}
{{define "GroupTemplate" -}}
{{- range .Grouped}}
### {{ .Name }}

{{range .Items -}}
* {{template "CommitTemplate" . }} {{.Title}}{{template "PullTemplate" . }}
{{end -}}
{{end -}}
{{end -}}
{{define "FlatTemplate" -}}
{{range .Items -}}
* {{template "CommitTemplate" . }} {{.Title}}{{template "PullTemplate" . }}
{{end -}}
{{end -}}
{{define "DefaultTemplate" -}}
{{if len .Grouped -}}
{{template "GroupTemplate" . -}}
{{- else}}
{{template "FlatTemplate" . -}}
{{end}}
<em>For more details, see <a href="{{.CompareURL}}">{{.PreviousVersion}}..{{.Version}}</a></em>
{{end -}}
{{template "DefaultTemplate" . -}}