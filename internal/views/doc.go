// Package views loads and executes layout-based HTML templates.
//
// A template root must contain layouts, partials, and pages directories. Each
// layout is a flat HTML file in layouts. Its partials live in the matching
// partials/<layout> directory, and its pages live in pages/<layout>.
// Both partials and pages may use nested subdirectories.
package views
