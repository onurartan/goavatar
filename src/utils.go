package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"net/http"
	"strings"
	"unicode"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"os"
	"strconv"
)

type GithubUser struct {
	Name string `json:"name"`
}

var loadedFont font.Face
var fontCache = make(map[int]font.Face)

func writeError(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "error",
		"message": message,
	})
}

func getFont(fontPath string, size int) font.Face {
	if face, ok := fontCache[size]; ok {
		return face
	}

	fontBytes, err := os.ReadFile(fontPath)
	if err != nil {
		log.Printf("Font okunamadı: %v", err)
		return nil
	}

	tt, err := opentype.Parse(fontBytes)
	if err != nil {
		log.Printf("Font parse hatası: %v", err)
		return nil
	}

	face, err := opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    float64(size),
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Printf("Font face oluşturulamadı: %v", err)
		return nil
	}

	fontCache[size] = face
	return face
}

func fetchGitHubName(username string) (string, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/users/%s", username))
	if err != nil {
		return "", fmt.Errorf("error while fetching GitHub user: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status: %d, unable to fetch user", resp.StatusCode)
	}

	var user GithubUser
	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		return "", fmt.Errorf("error parsing GitHub response: %v", err)
	}

	if user.Name == "" {
		user.Name = username
	}

	return user.Name, nil
}

func convertToInt(s string, objName string) (int, error) {
	result, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid number(%s): %s", objName, s)
	}
	return result, nil
}

func generateGradient(name string) (color.RGBA, color.RGBA) {
	hash := md5.Sum([]byte(name))
	color1 := color.RGBA{R: hash[0], G: hash[1], B: hash[2], A: 255}
	color2 := color.RGBA{R: hash[3], G: hash[4], B: hash[5], A: 255}
	return color1, color2
}

func getInitials(name string) string {
	var initials string
	for _, word := range strings.Fields(name) {
		if len(word) > 0 {
			initials += string(unicode.ToUpper(rune(word[0])))
		}
	}
	if len(initials) == 0 && len(name) > 0 {
		initials = string(unicode.ToUpper(rune(name[0])))
	}
	return initials
}

func getTextColor(bg color.RGBA) string {
	// >_ constrart for text color
	luminance := 0.299*float64(bg.R) + 0.587*float64(bg.G) + 0.114*float64(bg.B)
	if luminance > 186 {
		return "black"
	}
	return "white"
}

func determineTextColor(bg color.RGBA, input string) color.Color {
	switch strings.ToLower(input) {
	case "white":
		return color.White
	case "black":
		return color.Black
	default:
		if getTextColor(bg) == "black" {
			return color.Black
		}
		return color.White
	}
}

func generateSVG(size int, name string, color1, color2 color.RGBA, text string, rounded int, textColor string) string {
	if text == "auto" {
		text = getInitials(name)
	}

	fontSize := size / 3
	if len(text) > 1 {
		fontSize = size / 5
	}

	fill := "white"
	if len(textColor) > 0 {
		fill = textColor
	} else {
		fill = getTextColor(color1)
	}

	svgCode := ""

	if text != "" {
		svgCode = fmt.Sprintf(`<text x="50%%" y="50%%" text-anchor="middle" dominant-baseline="middle" font-family="sans-serif" font-size="%d" fill="%s">%s</text>`, fontSize, fill, text)
	}

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
	<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg">
		<defs>
			<linearGradient id="gradient" x1="1" y1="1" x2="0" y2="0">
				<stop offset="0%%" stop-color="rgb(%d,%d,%d)" />
				<stop offset="100%%" stop-color="rgb(%d,%d,%d)" />
			</linearGradient>
		</defs>
		<rect width="%d" height="%d" rx="%d" ry="%d" fill="url(#gradient)" />
		%s
	</svg>`, size, size, color1.R, color1.G, color1.B, color2.R, color2.G, color2.B,
		size, size, rounded, rounded, svgCode)
}

func drawText(img *image.RGBA, text string, textColor color.Color, size int) {
	col := textColor

	// if loadedFont == nil {
	fontSize := int(float64(size) / 3)
	loadedFont = getFont("fonts/Inter_24pt-Medium.ttf", fontSize)
	// }

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: loadedFont,
	}

	textWidth := d.MeasureString(text).Round()

	metrics := loadedFont.Metrics()
	ascent := metrics.Ascent.Ceil()
	descent := metrics.Descent.Ceil()
	textHeight := ascent + descent

	// >_ Postion(Center)
	x := (size - textWidth) / 2
	y := (size-textHeight)/2 + ascent

	d.Dot = fixed.P(x, y)
	d.DrawString(text)
}

func imageResponse(name string, w http.ResponseWriter, r *http.Request) {
	typeParam := r.URL.Query().Get("type")
	initials := r.URL.Query().Get("initials")
	color_QUERY := r.URL.Query().Get("color")
	width := r.URL.Query().Get("w")

	if initials == "auto" {
		initials = getInitials(name)
	}

	size := 120

	if width != "" {
		widthInt, convert_err := convertToInt(width, "width")
		if convert_err != nil {
			// http.Error(w, convert_err.Error(), http.StatusBadRequest)
			writeError(w, http.StatusBadRequest, convert_err.Error())
			return
		}

		// >_ To avoid overloading the server, I have limited the use of very large size values. This prevents malicious requests from crashing the system.
		if widthInt > 1080 {
			// http.Error(w, "width value cannot be greater than 1080", http.StatusBadRequest)
			writeError(w, http.StatusBadRequest, "Width value cannot be greater than 1080")
			return
		}

		size = widthInt
	}

	rounded := 0

	color1, color2 := generateGradient(name)
	textColor := determineTextColor(color1, color_QUERY)

	if typeParam == "svg" {
		svg := generateSVG(size, name, color1, color2, initials, rounded, color_QUERY)
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write([]byte(svg))
	} else {
		img := image.NewRGBA(image.Rect(0, 0, size, size))
		draw.Draw(img, img.Bounds(), &image.Uniform{color1}, image.Point{}, draw.Src)

		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				ratio := float64(x+y) / float64(2*size)
				c := color.RGBA{
					R: uint8(float64(color1.R)*(1-ratio) + float64(color2.R)*ratio),
					G: uint8(float64(color1.G)*(1-ratio) + float64(color2.G)*ratio),
					B: uint8(float64(color1.B)*(1-ratio) + float64(color2.B)*ratio),
					A: 255,
				}
				img.Set(x, y, c)
			}
		}

		if initials != "" {
			drawText(img, initials, textColor, size)
		}

		var buf bytes.Buffer
		err := png.Encode(&buf, img)
		if err != nil {
			// http.Error(w, "Failed to encode image", http.StatusInternalServerError)
			writeError(w, http.StatusInternalServerError, "Failed to encode image")
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write(buf.Bytes())
	}
}
