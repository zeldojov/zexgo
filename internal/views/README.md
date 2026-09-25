# views

`views` is the template loader and renderer used by the application. It
connects one layout with the partials and pages that belong to that layout.

## Contract

The filesystem passed to `New` must contain this structure:

```text
templates/
├── layouts/
│   ├── public.html
│   └── auth.html
├── partials/
│   ├── public/
│   │   ├── footer.html
│   │   └── navigation/
│   │       └── mobile.html
│   └── auth/
│       └── footer.html
└── pages/
    ├── public/
    │   ├── index.html
    │   └── admin/
    │       └── dashboard.html
    └── auth/
        └── login.html
```

The following rules are part of the package contract:

- `layouts` must exist and contain at least one `.html` layout file.
- Layout files must be directly inside `layouts`; nested layout files are not
  supported.
- Every layout must have a matching `partials/<layout>` directory.
- Every layout must have a matching `pages/<layout>` directory.
- Every layout partial directory must contain at least one `.html` file.
- Every layout pages directory must contain at least one `.html` file.
- Partials and pages are discovered recursively below their layout directory.
- Files directly inside `partials` or `pages` are invalid.
- A partial or pages directory whose layout does not exist is invalid.
- A layout must define `layout`.
- A page must define `content`.
- Non-HTML files are ignored below layout, partial, and page directories.

The template root is interpreted relative to the supplied `fs.FS`. For an
embedded filesystem created with:

```go
//go:embed templates
var templateFS embed.FS
```

the template root is normally `templates`.

## Template definitions

A layout defines the entry point for rendering:

```gotemplate
{{ define "layout" }}
<!doctype html>
<html>
<body>
    {{ template "content" . }}
    {{ template "footer" . }}
</body>
</html>
{{ end }}
```

A page defines the content inserted by its layout:

```gotemplate
{{ define "content" }}
<h1>{{ .Title }}</h1>
{{ end }}
```

A partial is defined by name and can be used by the matching layout or its
pages:

```gotemplate
{{ define "footer" }}
<footer>zexgo</footer>
{{ end }}
```

Partial names come from `define`, not from the partial file path. Files in
different layout directories are parsed into separate template instances, so
both `partials/public/footer.html` and `partials/auth/footer.html` may define
`footer` without sharing a namespace.

## Registered page names

Page names are derived from the page path relative to the `pages` directory,
with the `.html` extension removed:

```text
pages/public/index.html
    -> public/index

pages/public/admin/dashboard.html
    -> public/admin/dashboard
```

Render a page with its registered name:

```go
var output bytes.Buffer

err := views.ExecuteTemplate(&output, "public/admin/dashboard", data)
if err != nil {
    log.Fatal(err)
}
```

The filesystem path is only used to discover and parse the file. The name
passed to `ExecuteTemplate` is the registered page name above.

## Initialization

`New` parses every layout and page during application startup. Template
functions must be supplied before parsing:

```go
views, err := viewspkg.New(templateFS, "templates", template.FuncMap{
    "upper": strings.ToUpper,
})
if err != nil {
    log.Fatal(err)
}
```

A function used by a template but missing from the `FuncMap` causes
initialization to fail.

## Execution and errors

`ExecuteTemplate` writes rendered HTML to the supplied `io.Writer`. It does not
know about HTTP and does not set response headers. HTTP handlers should decide
how to set headers and handle the returned error.

The package exposes two sentinel errors:

```go
errors.Is(err, views.ErrTemplateWalk)
errors.Is(err, views.ErrTemplateParse)
```

`ErrTemplateWalk` identifies failures while recursively walking page or
partial template directories. Filesystem errors raised while initially
reading or validating the root directories are returned as ordinary wrapped
errors. `ErrTemplateParse` identifies template parsing and layout-cloning
failures.
Other validation and execution errors include context in their messages and
are returned directly to the caller.

A typical HTTP handler can render into a buffer before writing the response:

```go
var output bytes.Buffer
if err := views.ExecuteTemplate(&output, "public/index", data); err != nil {
    log.Printf("render template: %v", err)
    http.Error(w, "Internal Server Error", http.StatusInternalServerError)
    return
}

w.Header().Set("Content-Type", "text/html; charset=utf-8")
_, _ = w.Write(output.Bytes())
```

The `Views` instance should be initialized once and reused. After successful
initialization, its parsed templates are read-only and may be executed for
multiple requests.

## Tests

The package tests cover valid nested pages, recursive partials, template
functions, isolated layout namespaces, invalid directory structures, missing
required definitions, parser and walker errors, runtime execution errors, and
invalid `ExecuteTemplate` arguments.
