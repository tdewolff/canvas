#!/usr/bin/bash
go run -tags harfbuzz,fribidi main.go
latexmk preview.tex
