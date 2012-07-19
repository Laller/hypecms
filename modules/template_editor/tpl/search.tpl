{{require admin/header.t}}
{{require template_editor/sidebar.t}}

{{if .error}}
	There was an error: {{.error}}
{{else}}
	{{$is_pub := .is_public}}
	{{$is_priv := .is_private}}
	{{$is_mod := .is_mod}}
	
	{{range .dir}}
		{{if $is_pub}}
			<a href="/admin/template_editor/view/public/{{.Name}}?file=">{{.Name}}</a><br />
		{{end}}
		
		{{if $is_priv}}
			<a href="/admin/template_editor/view/private/{{.Name}}?file=">{{.Name}}</a><br />
		{{end}}
		
		{{if $is_mod}}
			<a href="/admin/template_editor/view/mod/{{.Name}}?file=">{{.Name}}</a><br />
		{{end}}
	{{end}}
{{end}}

{{require template_editor/footer.t}}
{{require admin/footer.t}}