package views

import (
	"bytes"
	"embed"
	"errors"
	"html/template"
	"io/fs"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
)

//go:embed testdata/templates
var embeddedTemplates embed.FS

func validFS() fstest.MapFS {
	return fstest.MapFS{
		"templates/layouts/public.html": &fstest.MapFile{
			Data: []byte(`
                {{ define "layout" }}
                    {{ template "content" . }}
                    {{ template "footer" . }}
                {{ end }}
            `),
		},
		"templates/partials/public/footer.html": &fstest.MapFile{
			Data: []byte(`
                {{ define "footer" }}footer{{ end }}
            `),
		},
		"templates/pages/public/admin/index.html": &fstest.MapFile{
			Data: []byte(`
                {{ define "content" }}Hello {{ .Name }}{{ end }}
            `),
		},
	}
}

func TestNewAndExecuteNestedPage(t *testing.T) {
	views, err := New(validFS(), "templates", nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	var output bytes.Buffer

	err = views.ExecuteTemplate(
		&output,
		"public/admin/index",
		struct {
			Name string
		}{
			Name: "Željko",
		},
	)
	if err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}

	if !strings.Contains(output.String(), "Hello Željko") {
		t.Fatalf("output = %q, expected rendered page", output.String())
	}

	if !strings.Contains(output.String(), "footer") {
		t.Fatalf("output = %q, expected rendered partial", output.String())
	}
}

func TestNewAndExecuteEmbeddedTemplates(t *testing.T) {
	views, err := New(embeddedTemplates, "testdata/templates", nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	var output bytes.Buffer
	if err := views.ExecuteTemplate(&output, "public/index", struct {
		Name string
	}{Name: "embedded"}); err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}

	for _, expected := range []string{"embedded page", "embedded footer"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("output = %q, expected %q", output.String(), expected)
		}
	}
}

func TestNewAndExecuteConcurrentTemplates(t *testing.T) {
	views, err := New(validFS(), "templates", nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	const workers = 32
	errorsCh := make(chan error, workers)
	var waitGroup sync.WaitGroup
	waitGroup.Add(workers)

	for worker := 0; worker < workers; worker++ {
		go func() {
			defer waitGroup.Done()

			var output bytes.Buffer
			if err := views.ExecuteTemplate(&output, "public/admin/index", struct {
				Name string
			}{Name: "concurrent"}); err != nil {
				errorsCh <- err
				return
			}

			if !strings.Contains(output.String(), "Hello concurrent") {
				errorsCh <- errors.New("concurrent render output is incomplete")
			}
		}()
	}

	waitGroup.Wait()
	close(errorsCh)

	for err := range errorsCh {
		t.Errorf("concurrent ExecuteTemplate() error = %v", err)
	}
}

func TestExecuteTemplateUnknownPage(t *testing.T) {
	views, err := New(validFS(), "templates", nil)
	if err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer

	err = views.ExecuteTemplate(&output, "public/missing", nil)
	if err == nil {
		t.Fatal("expected error for unknown page")
	}
}

func TestNewAndExecuteRecursivePartialWithFuncMap(t *testing.T) {
	fsys := validFSWithFile(
		"templates/layouts/public.html",
		`{{ define "layout" }}{{ template "content" . }}{{ template "footer" . }}{{ template "mobile-nav" . }}{{ end }}`,
	)
	fsys = validFSWithFileOn(
		fsys,
		"templates/partials/public/navigation/mobile.html",
		`{{ define "mobile-nav" }}mobile navigation{{ end }}`,
	)
	fsys = validFSWithFileOn(
		fsys,
		"templates/pages/public/admin/index.html",
		`{{ define "content" }}{{ upper .Name }}{{ end }}`,
	)

	views, err := New(fsys, "templates", template.FuncMap{
		"upper": strings.ToUpper,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	var output bytes.Buffer
	if err := views.ExecuteTemplate(&output, "public/admin/index", struct {
		Name string
	}{Name: "Željko"}); err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}

	for _, expected := range []string{"ŽELJKO", "footer", "mobile navigation"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("output = %q, expected %q", output.String(), expected)
		}
	}
}

func TestNewIgnoresNonHTMLFiles(t *testing.T) {
	fsys := validFSWithFileOn(
		validFS(),
		"templates/layouts/README.txt",
		"layout documentation",
	)
	fsys = validFSWithFileOn(fsys, "templates/partials/public/README.txt", "partial documentation")
	fsys = validFSWithFileOn(fsys, "templates/pages/public/README.txt", "page documentation")

	if _, err := New(fsys, "templates", nil); err != nil {
		t.Fatalf("New() error = %v, non-HTML files should be ignored", err)
	}
}

func TestNewRequiresMatchingDirectoriesForEveryLayout(t *testing.T) {
	tests := []struct {
		name   string
		remove string
		want   string
	}{
		{
			name:   "partials",
			remove: "templates/partials/auth/footer.html",
			want:   `partials/auth`,
		},
		{
			name:   "pages",
			remove: "templates/pages/auth/login.html",
			want:   `pages/auth`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fsys := validTwoLayoutFS()
			delete(fsys, test.remove)

			_, err := New(fsys, "templates", nil)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("New() error = %v, want path containing %q", err, test.want)
			}
		})
	}
}

func TestLayoutsHaveIsolatedPartialNamespaces(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/layouts/public.html": &fstest.MapFile{Data: []byte(
			`{{ define "layout" }}{{ template "content" . }}{{ template "footer" . }}{{ end }}`,
		)},
		"templates/layouts/auth.html": &fstest.MapFile{Data: []byte(
			`{{ define "layout" }}{{ template "content" . }}{{ template "footer" . }}{{ end }}`,
		)},
		"templates/partials/public/footer.html": &fstest.MapFile{Data: []byte(
			`{{ define "footer" }}public footer{{ end }}`,
		)},
		"templates/partials/auth/footer.html": &fstest.MapFile{Data: []byte(
			`{{ define "footer" }}auth footer{{ end }}`,
		)},
		"templates/pages/public/index.html": &fstest.MapFile{Data: []byte(
			`{{ define "content" }}public page{{ end }}`,
		)},
		"templates/pages/auth/login.html": &fstest.MapFile{Data: []byte(
			`{{ define "content" }}auth page{{ end }}`,
		)},
	}

	views, err := New(fsys, "templates", nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	for _, test := range []struct {
		name   string
		footer string
	}{
		{name: "public/index", footer: "public footer"},
		{name: "auth/login", footer: "auth footer"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := views.ExecuteTemplate(&output, test.name, nil); err != nil {
				t.Fatalf("ExecuteTemplate() error = %v", err)
			}

			if !strings.Contains(output.String(), test.footer) {
				t.Fatalf("output = %q, expected %q", output.String(), test.footer)
			}
		})
	}
}

func TestExecuteTemplateValidation(t *testing.T) {
	views, err := New(validFS(), "templates", nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	var nilViews *Views
	tests := []struct {
		name string
		call func() error
		want string
	}{
		{
			name: "nil receiver",
			call: func() error {
				return nilViews.ExecuteTemplate(&bytes.Buffer{}, "public/admin/index", nil)
			},
			want: "views are not initialized",
		},
		{
			name: "uninitialized views",
			call: func() error {
				return (&Views{}).ExecuteTemplate(&bytes.Buffer{}, "public/admin/index", nil)
			},
			want: "views are not initialized",
		},
		{
			name: "nil writer",
			call: func() error {
				return views.ExecuteTemplate(nil, "public/admin/index", nil)
			},
			want: "template writer is nil",
		},
		{
			name: "empty name",
			call: func() error {
				return views.ExecuteTemplate(&bytes.Buffer{}, "", nil)
			},
			want: "template name is empty",
		},
		{
			name: "unknown name",
			call: func() error {
				return views.ExecuteTemplate(&bytes.Buffer{}, "public/missing", nil)
			},
			want: "is not registered",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.call()
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestNewRejectsNilFilesystem(t *testing.T) {
	views, err := New(nil, "templates", nil)
	if views != nil {
		t.Fatalf("New() views = %#v, want nil", views)
	}
	if err == nil || err.Error() != "template filesystem is nil" {
		t.Fatalf("New() error = %v, want nil filesystem error", err)
	}
}

func TestExecuteTemplateReturnsRuntimeError(t *testing.T) {
	fsys := validFSWithFile(
		"templates/pages/public/admin/index.html",
		`{{ define "content" }}{{ fail }}{{ end }}`,
	)
	views, err := New(fsys, "templates", template.FuncMap{
		"fail": func() (string, error) {
			return "", errors.New("expected runtime failure")
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = views.ExecuteTemplate(&bytes.Buffer{}, "public/admin/index", nil)
	if err == nil || !strings.Contains(err.Error(), "expected runtime failure") {
		t.Fatalf("ExecuteTemplate() error = %v, want runtime failure", err)
	}
}

func TestExecuteTemplateReturnsWriterError(t *testing.T) {
	views, err := New(validFS(), "templates", nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = views.ExecuteTemplate(errorWriter{}, "public/admin/index", nil)
	if err == nil || !strings.Contains(err.Error(), "expected writer failure") {
		t.Fatalf("ExecuteTemplate() error = %v, want writer failure", err)
	}
}

func TestNewWrapsTemplateErrors(t *testing.T) {
	parseFS := validFSWithFile(
		"templates/layouts/public.html",
		`{{ define "layout" }}`,
	)
	_, err := New(parseFS, "templates", nil)
	if !errors.Is(err, ErrTemplateParse) {
		t.Fatalf("New() error = %v, want ErrTemplateParse", err)
	}

	walkFS := validFSWithout("templates/partials/public/footer.html")
	_, err = New(walkFS, "templates", nil)
	if !errors.Is(err, ErrTemplateWalk) {
		t.Fatalf("New() error = %v, want ErrTemplateWalk", err)
	}
}

func TestNewRejectsInvalidTemplateStructure(t *testing.T) {
	tests := []struct {
		name string
		fsys fstest.MapFS
		want string
	}{
		{
			name: "layouts directory does not exist",
			fsys: fstest.MapFS{
				"templates/pages/public/index.html": &fstest.MapFile{
					Data: []byte(`{{ define "content" }}index{{ end }}`),
				},
			},
			want: "read layouts directory",
		},
		{
			name: "layouts directory has no HTML files",
			fsys: validFSWithFileOn(
				validFSWithout("templates/layouts/public.html"),
				"templates/layouts/README.txt",
				"layout documentation",
			),
			want: "no layouts found",
		},
		{
			name: "layouts directory is not flat",
			fsys: validFSWithDirectory(
				"templates/layouts/admin",
				"templates/layouts/admin/public.html",
			),
			want: "layouts directory must be flat",
		},
		{
			name: "partials directory does not exist",
			fsys: validFSWithout("templates/partials/public/footer.html"),
			want: "templates/partials/public",
		},
		{
			name: "layout partial directory does not exist",
			fsys: validFSWithout("templates/partials/public/footer.html"),
			want: "templates/partials/public",
		},
		{
			name: "layout partial directory is empty",
			fsys: validFSWithDirectory(
				"templates/partials/public",
				"templates/partials/public/footer.html",
			),
			want: "no partial templates found",
		},
		{
			name: "pages directory does not exist",
			fsys: validFSWithout("templates/pages/public/admin/index.html"),
			want: "pages directory",
		},
		{
			name: "layout pages directory is empty",
			fsys: validFSWithDirectory(
				"templates/pages/public",
				"templates/pages/public/admin/index.html",
			),
			want: "no page templates found",
		},
		{
			name: "page does not define content",
			fsys: validFSWithFile(
				"templates/pages/public/index.html",
				`{{ define "page" }}index{{ end }}`,
			),
			want: `does not define "content"`,
		},
		{
			name: "layout does not define layout",
			fsys: validFSWithFile(
				"templates/layouts/public.html",
				`{{ define "base" }}base{{ end }}`,
			),
			want: `template "layout" is not defined`,
		},
		{
			name: "invalid template syntax",
			fsys: validFSWithFile(
				"templates/layouts/public.html",
				`{{ define "layout" }}`,
			),
			want: "template parse error",
		},
		{
			name: "invalid partial syntax",
			fsys: validFSWithFile(
				"templates/partials/public/footer.html",
				`{{ define "footer" }`,
			),
			want: "template parse error",
		},
		{
			name: "invalid page syntax",
			fsys: validFSWithFile(
				"templates/pages/public/admin/index.html",
				`{{ define "content" }`,
			),
			want: "template parse error",
		},
		{
			name: "missing template function",
			fsys: validFSWithFile(
				"templates/pages/public/admin/index.html",
				`{{ define "content" }}{{ missing . }}{{ end }}`,
			),
			want: "template parse error",
		},
		{
			name: "file directly in pages directory",
			fsys: validFSWithFile(
				"templates/pages/index.html",
				`{{ define "content" }}index{{ end }}`,
			),
			want: "unexpected file in pages directory",
		},
		{
			name: "file directly in partials directory",
			fsys: validFSWithFile(
				"templates/partials/orphan.html",
				`{{ define "orphan" }}orphan{{ end }}`,
			),
			want: "unexpected file in partials directory",
		},
		{
			name: "partials layout has no matching layout",
			fsys: validFSWithDirectory("templates/partials/auth"),
			want: `no layout found for partials directory "auth"`,
		},
		{
			name: "pages layout has no matching layout",
			fsys: validFSWithDirectory("templates/pages/auth"),
			want: `no layout found for pages directory "auth"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := New(test.fsys, "templates", nil)
			if err == nil {
				t.Fatal("New() error = nil, want error")
			}

			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("New() error = %q, want substring %q", err, test.want)
			}
		})
	}
}

func validFSWithout(path string) fstest.MapFS {
	fsys := validFS()
	delete(fsys, path)
	return fsys
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) {
	return 0, errors.New("expected writer failure")
}

func validTwoLayoutFS() fstest.MapFS {
	fsys := validFS()
	fsys = validFSWithFileOn(fsys, "templates/layouts/auth.html", `{{ define "layout" }}{{ template "content" . }}{{ template "footer" . }}{{ end }}`)
	fsys = validFSWithFileOn(fsys, "templates/partials/auth/footer.html", `{{ define "footer" }}auth footer{{ end }}`)
	return validFSWithFileOn(fsys, "templates/pages/auth/login.html", `{{ define "content" }}auth page{{ end }}`)
}

func validFSWithFile(path, content string) fstest.MapFS {
	fsys := validFS()
	return validFSWithFileOn(fsys, path, content)
}

func validFSWithFileOn(fsys fstest.MapFS, path, content string) fstest.MapFS {
	fsys[path] = &fstest.MapFile{Data: []byte(content)}
	return fsys
}

func validFSWithDirectory(path string, remove ...string) fstest.MapFS {
	fsys := validFS()
	for _, file := range remove {
		delete(fsys, file)
	}
	fsys[path] = &fstest.MapFile{Mode: fs.ModeDir}
	return fsys
}
