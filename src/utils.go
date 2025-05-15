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
	"sync"
)

var (
	parsedFont *opentype.Font
	fontCache  = make(map[int]font.Face)
	fontMu     sync.Mutex
)

type GithubUser struct {
	Name string `json:"name"`
}

// ######### Google Colors ######### //
var googleColors = []color.RGBA{
	{R: 244, G: 67, B: 54, A: 255},  // Red
	{R: 233, G: 30, B: 99, A: 255},  // Pink
	{R: 156, G: 39, B: 176, A: 255}, // Purple
	{R: 103, G: 58, B: 183, A: 255}, // Deep Purple
	{R: 63, G: 81, B: 181, A: 255},  // Indigo
	{R: 33, G: 150, B: 243, A: 255}, // Blue
	{R: 3, G: 169, B: 244, A: 255},  // Light Blue
	{R: 0, G: 188, B: 212, A: 255},  // Cyan
	{R: 0, G: 150, B: 136, A: 255},  // Teal
	{R: 76, G: 175, B: 80, A: 255},  // Green
	{R: 139, G: 195, B: 74, A: 255}, // Light Green
	{R: 205, G: 220, B: 57, A: 255}, // Lime
	// {R: 255, G: 235, B: 59, A: 255},  // Yellow
	{R: 255, G: 193, B: 7, A: 255},   // Amber
	{R: 255, G: 152, B: 0, A: 255},   // Orange
	{R: 255, G: 87, B: 34, A: 255},   // Deep Orange
	{R: 121, G: 85, B: 72, A: 255},   // Brown
	{R: 158, G: 158, B: 158, A: 255}, // Grey
	{R: 96, G: 125, B: 139, A: 255},  // Blue Grey
	{R: 0, G: 121, B: 107, A: 255},   // Custom Teal
	{R: 85, G: 139, B: 47, A: 255},   // Olive Green
}

// var fontCache = make(map[int]font.Face) // *old version
func loadFontOnce(fontPath string) error {
	fontMu.Lock()
	defer fontMu.Unlock()
	if parsedFont != nil {
		return nil
	}
	fontBytes, err := os.ReadFile(fontPath)
	if err != nil {
		return fmt.Errorf("failed to read font file: %w", err)
	}
	parsedFont, err = opentype.Parse(fontBytes)
	if err != nil {
		return fmt.Errorf("failed to parse font file: %w", err)
	}
	return nil
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "error",
		"message": message,
	})
}

func getFont(fontPath string, size int) font.Face {
	fontMu.Lock()
	defer fontMu.Unlock()

	if face, ok := fontCache[size]; ok {
		return face
	}

	if parsedFont == nil {
		return nil
	}

	face, err := opentype.NewFace(parsedFont, &opentype.FaceOptions{
		Size:    float64(size),
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Printf("failed to create font face for size %d: %v", size, err)
		return nil
	}

	fontCache[size] = face
	return face
}

func fetchGitHubName(username string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/users/%s", username)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("User-Agent", "goavatar-app")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error while fetching GitHub user: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden && resp.Header.Get("X-RateLimit-Remaining") == "0" {
			return "", fmt.Errorf("GitHub API rate limit exceeded")
		}
		return "", fmt.Errorf("GitHub API returned status: %d", resp.StatusCode)
	}

	var user GithubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
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

func getColorFromPalette(name string) color.RGBA {
	hash := md5.Sum([]byte(name))
	index := int(hash[0]) % len(googleColors)
	return googleColors[index]
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

func generateSVG(size int, name string, color1, color2 color.RGBA, text string, rounded int, textColor string, aType string) string {

	if aType == "" {
		aType = "gradient"
	}

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

	if aType == "color" {

		return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
	<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg">
		<rect width="%d" height="%d" rx="%d" ry="%d" fill="rgb(%d,%d,%d)" />
		%s
	</svg>`, size, size, size, size, rounded, rounded, color1.R, color1.G, color1.B, svgCode)
	} else {
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

}

func drawText(img *image.RGBA, text string, textColor color.Color, size int) {
	col := textColor

	fontSize := int(float64(size) / 2)
	loadedFont := getFont("fonts/Inter_24pt-Medium.ttf", fontSize)

	if loadedFont == nil {
		log.Println("Font failed to load. Unable to draw text.")
		return
	}

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

func imageResponse(name string, w http.ResponseWriter, r *http.Request, _initialsActivate bool) {
	typeParam := r.URL.Query().Get("type")
	initials := r.URL.Query().Get("initials")
	color_QUERY := r.URL.Query().Get("color")
	iName := r.URL.Query().Get("iName")
	aType := r.URL.Query().Get("aType") // image bg gradient or color default=gradient
	width := r.URL.Query().Get("w")

	if aType != "color" && aType != "gradient" {
		aType = "gradient"
	}

	if _initialsActivate {
		initials = "auto"
	}

	if initials == "auto" {
		initials_name := name
		if iName != "" {
			initials_name = iName
		}
		initials = getInitials(initials_name)
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

	var color1, color2 color.RGBA

	if aType == "color" {
		color1 = getColorFromPalette(name)
		color2 = color1
	} else {
		color1, color2 = generateGradient(name)
	}

	textColor := determineTextColor(color1, color_QUERY)

	rounded := 0 //

	if typeParam == "svg" {
		svg := generateSVG(size, name, color1, color2, initials, rounded, color_QUERY, aType)
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "public, max-age=43200")
		w.Write([]byte(svg))
	} else {
		img := image.NewRGBA(image.Rect(0, 0, size, size))
		draw.Draw(img, img.Bounds(), &image.Uniform{color1}, image.Point{}, draw.Src)

		if aType == "gradient" {
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
		w.Header().Set("Cache-Control", "public, max-age=43200")
		w.Write(buf.Bytes())
	}
}
