package grades

import (
	"html/template"
)

// commentData is the template model for buildComment.
type commentData struct {
	Comment string
	Files   []uploadedFile
}

// Label returns a human-readable title for the file column.
// Defined as a method so it's directly callable from the template.
func (f uploadedFile) Label() string {
	switch f.Column {
	case "pdf_exam_file1":
		return "Exam Submission — Format 1"
	case "pdf_exam_file2":
		return "Exam Submission — Format 2"
	case "pdf_eval_file":
		return "Detailed Feedback"
	default:
		return "Feedback File"
	}
}

// commentTmpl is parsed once at package init — zero allocation on every call.
var commentTmpl = template.Must(template.New("comment").Parse(`
{{- if .Comment -}}
<p>{{ .Comment }}</p>
{{ end -}}
{{- range .Files }}
<div style="margin:8px 0;padding:8px 12px;border-left:3px solid #0375d8;background:#f5f9ff;">
  <p style="margin:0 0 4px;font-weight:bold;font-size:13px;">{{ .Label }}</p>
  <p style="margin:0;font-size:12px;color:#555;">{{ .Name }}</p>
  <p style="margin:4px 0 0;">
    <a href="{{ .URL }}" target="_blank" style="margin-right:12px;">&#128196; View</a>
    <a href="{{ .DownloadURL }}">&#11015; Download</a>
  </p>
</div>
{{ end }}`))
