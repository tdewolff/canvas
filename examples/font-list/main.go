package main

import (
	"fmt"
	"github.com/tdewolff/canvas/font"
	"log"
	"sort"
	"strings"
)

func StyleString(style font.Style) string {
	switch style {
	case font.UnknownStyle:
		return "Unknown"
	case font.Regular:
		return "Regular"
	case font.Thin:
		return "Thin"
	case font.ExtraLight:
		return "ExtraLight"
	case font.Light:
		return "Light"
	case font.Medium:
		return "Medium"
	case font.SemiBold:
		return "SemiBold"
	case font.Bold:
		return "Bold"
	case font.ExtraBold:
		return "ExtraBold"
	case font.Black:
		return "Black"
	case font.Italic:
		return "Italic"
	}
	return ""
}

// Finds and lists the default system fonts
func main() {
	var fonts *font.SystemFonts
	dirs := font.DefaultFontDirs()
	fonts, err := font.FindSystemFonts(dirs)
	if err != nil {
		log.Fatalf("Could not find system fonts: %v", err)
	}
	// list default system fonts
	fmt.Println("Default Fonts by Category:\n")
	for category, items := range fonts.Defaults {
		fmt.Println(category)
		for _, item := range items {
			fmt.Println("  ", item)
		}
	}
	// list all fonts
	var resultStrings []string
	for category, styleMap := range fonts.Fonts {
		var styles []font.Style
		for _, metadata := range styleMap {
			styles = append(styles, metadata.Style)
		}
		sort.SliceStable(styles, func(i, j int) bool {
			return styles[i] < styles[j]
		})
		var styleNames []string
		for _, style := range styles {
			styleNames = append(styleNames, StyleString(style))
		}
		fontStr := fmt.Sprintf("%s: [%s]", category, strings.Join(styleNames, ", "))
		resultStrings = append(resultStrings, fontStr)
	}
	sort.Strings(resultStrings)
	fmt.Println("\n\nAll Fonts\nFont Family: [Available Styles]\n")
	for _, fontStr := range resultStrings {
		fmt.Println(fontStr)
	}
}
