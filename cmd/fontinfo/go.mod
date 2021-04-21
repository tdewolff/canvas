module github.com/tdewolff/canvas/cmd/fontinfo

go 1.15

replace github.com/tdewolff/canvas => ../../

replace github.com/tdewolff/argp => ../../../argp

require (
	github.com/tdewolff/argp v0.0.0-00010101000000-000000000000
	github.com/tdewolff/canvas v0.0.0-00010101000000-000000000000
	golang.org/x/image v0.0.0-20210220032944-ac19c3e999fb
)
