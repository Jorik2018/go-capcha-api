package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"
	"bytes"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
"golang.org/x/image/font"
"golang.org/x/image/font/opentype"
"golang.org/x/image/font/gofont/gobold"
"golang.org/x/image/math/fixed"
)
import "encoding/base64"
var (
	ctx = context.Background()

	redisClient *redis.Client
)

const (
	captchaTTL    = 3 * time.Minute
	captchaLength = 5
)

const captchaChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

type CaptchaValidationRequest struct {
	CaptchaID string `json:"captchaId"`
	Captcha   string `json:"captcha"`
}

func main() {
	redisAddr := getEnv("REDIS_HOST", "localhost:6379")
	port := getEnv("PORT", "8080")

	redisClient = redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Could not connect to Redis at %s: %v", redisAddr, err)
	}

	http.HandleFunc("/health", healthHandler)

	http.HandleFunc("/new", createCaptchaHandler)
	http.HandleFunc("/validate", validateCaptchaHandler)
	http.HandleFunc("/", captchaImageHandler)

	addr := ":" + port

	fmt.Println("==========================================")
	fmt.Println("go-capcha-api")
	fmt.Println("==========================================")
	fmt.Println("Port: ", port)
	fmt.Println("Redis:", redisAddr)
	fmt.Println("==========================================")

	log.Fatal(http.ListenAndServe(addr, nil))
}

func healthHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	if err := redisClient.Ping(ctx).Err(); err != nil {
		writeJSON(
			w,
			http.StatusServiceUnavailable,
			map[string]any{
				"status": "error",
				"redis":  "unavailable",
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		map[string]any{
			"status": "ok",
			"redis":  "ok",
		},
	)
}

func createCaptchaHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	captchaID := uuid.New().String()
	code := generateCode(captchaLength)

	key := captchaKey(captchaID)

	err := redisClient.Set(
		ctx,
		key,
		code,
		captchaTTL,
	).Err()

	if err != nil {
		log.Println("Redis SET error:", err)

		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "Could not create captcha",
			},
		)

		return
	}

	img := generateCaptchaImage(code)

var buffer bytes.Buffer

if err := png.Encode(&buffer, img); err != nil {
	log.Println("Captcha PNG encode error:", err)

	redisClient.Del(ctx, key)

	writeJSON(
		w,
		http.StatusInternalServerError,
		map[string]string{
			"error": "Could not generate captcha image",
		},
	)

	return
}

imageBase64 := base64.StdEncoding.EncodeToString(
	buffer.Bytes(),
)

	writeJSON(
	w,
	http.StatusOK,
	map[string]any{
		"captchaId": captchaID,
		"image": "data:image/png;base64," +
			imageBase64,
		"expiresIn": int(captchaTTL.Seconds()),
	},
)
}

func captchaImageHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	captchaID := strings.TrimPrefix(
		r.URL.Path,
		"/api/captcha/",
	)

	captchaID = strings.TrimSpace(captchaID)

	if captchaID == "" {
		http.NotFound(w, r)
		return
	}

	// Evita que /validate entre aquí si cambia el routing
	if captchaID == "validate" || captchaID == "new" {
		http.NotFound(w, r)
		return
	}

	key := captchaKey(captchaID)

	code, err := redisClient.Get(
		ctx,
		key,
	).Result()

	if err == redis.Nil {
		writeJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": "Captcha expired or not found",
			},
		)

		return
	}

	if err != nil {
		log.Println("Redis GET error:", err)

		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "Could not read captcha",
			},
		)

		return
	}

	img := generateCaptchaImage(code)

	w.Header().Set(
		"Content-Type",
		"image/png",
	)

	w.Header().Set(
		"Cache-Control",
		"no-store, no-cache, must-revalidate, max-age=0",
	)

	w.Header().Set(
		"Pragma",
		"no-cache",
	)

	w.Header().Set(
		"Expires",
		"0",
	)

	if err := png.Encode(w, img); err != nil {
		log.Println("PNG encode error:", err)
	}
}

func validateCaptchaHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	var request CaptchaValidationRequest

	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&request); err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]any{
				"valid": false,
				"error": "Invalid JSON",
			},
		)

		return
	}

	request.CaptchaID =
		strings.TrimSpace(request.CaptchaID)

	request.Captcha =
		strings.TrimSpace(request.Captcha)

	if request.CaptchaID == "" ||
		request.Captcha == "" {

		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]any{
				"valid": false,
				"error": "captchaId and captcha are required",
			},
		)

		return
	}

	key := captchaKey(request.CaptchaID)

	expected, err := redisClient.GetDel(
		ctx,
		key,
	).Result()

	if err == redis.Nil {
		writeJSON(
			w,
			http.StatusOK,
			map[string]bool{
				"valid": false,
			},
		)

		return
	}

	if err != nil {
		log.Println("Redis GETDEL error:", err)

		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]any{
				"valid": false,
				"error": "Could not validate captcha",
			},
		)

		return
	}

	valid := strings.EqualFold(
		expected,
		request.Captcha,
	)

	writeJSON(
		w,
		http.StatusOK,
		map[string]bool{
			"valid": valid,
		},
	)
}

func generateCode(length int) string {
	var result strings.Builder

	result.Grow(length)

	for i := 0; i < length; i++ {
		result.WriteByte(
			captchaChars[
				secureRandomInt(
					len(captchaChars),
				),
			],
		)
	}

	return result.String()
}
func generateCaptchaImage(
	code string,
) image.Image {
	const (
		width  = 220
		height = 80
	)

	img := image.NewRGBA(
		image.Rect(
			0,
			0,
			width,
			height,
		),
	)

	draw.Draw(
		img,
		img.Bounds(),
		&image.Uniform{
			C: color.White,
		},
		image.Point{},
		draw.Src,
	)

	// Menos ruido para priorizar legibilidad
	for i := 0; i < 10; i++ {
		drawLine(
			img,
			secureRandomInt(width),
			secureRandomInt(height),
			secureRandomInt(width),
			secureRandomInt(height),
			color.RGBA{
				R: uint8(secureRandomInt(180)),
				G: uint8(secureRandomInt(180)),
				B: uint8(secureRandomInt(180)),
				A: 255,
			},
			3,
		)
	}

	ttf, err := opentype.Parse(gobold.TTF)
	if err != nil {
		panic(err)
	}

	face, err := opentype.NewFace(
		ttf,
		&opentype.FaceOptions{
			Size:    46,
			DPI:     72,
			Hinting: font.HintingFull,
		},
	)
	if err != nil {
		panic(err)
	}

	defer face.Close()

	x := 10

	for _, char := range code {
		y := 40 + secureRandomInt(12)

		d := &font.Drawer{
			Dst: img,

			Src: image.NewUniform(
				color.RGBA{
					R: uint8(30 + secureRandomInt(80)),
					G: uint8(30 + secureRandomInt(80)),
					B: uint8(30 + secureRandomInt(80)),
					A: 255,
				},
			),

			Face: face,

			Dot: fixed.Point26_6{
				X: fixed.I(x),
				Y: fixed.I(y),
			},
		}

		d.DrawString(string(char))

		x += 40
	}

	return img
}
func drawLine(
	img *image.RGBA,
	x0 int,
	y0 int,
	x1 int,
	y1 int,
	c color.Color,
	thickness int,
) {
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)

	sx := -1
	if x0 < x1 {
		sx = 1
	}

	sy := -1
	if y0 < y1 {
		sy = 1
	}

	err := dx + dy

	radius := thickness / 2

	for {
		// Pintar un círculo en vez de un solo píxel
		for ox := -radius; ox <= radius; ox++ {
			for oy := -radius; oy <= radius; oy++ {
				if ox*ox+oy*oy <= radius*radius {
					x := x0 + ox
					y := y0 + oy

					if image.Pt(x, y).In(img.Bounds()) {
						img.Set(x, y, c)
					}
				}
			}
		}

		if x0 == x1 && y0 == y1 {
			break
		}

		e2 := 2 * err

		if e2 >= dy {
			err += dy
			x0 += sx
		}

		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func secureRandomInt(
	max int,
) int {
	if max <= 0 {
		return 0
	}

	n, err := rand.Int(
		rand.Reader,
		big.NewInt(int64(max)),
	)

	if err != nil {
		panic(err)
	}

	return int(n.Int64())
}

func captchaKey(
	captchaID string,
) string {
	return "captcha:" + captchaID
}

func writeJSON(
	w http.ResponseWriter,
	status int,
	value any,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Println(
			"JSON encode error:",
			err,
		)
	}
}

func methodNotAllowed(
	w http.ResponseWriter,
) {
	writeJSON(
		w,
		http.StatusMethodNotAllowed,
		map[string]string{
			"error": "Method not allowed",
		},
	)
}

func getEnv(
	name string,
	defaultValue string,
) string {
	value := strings.TrimSpace(
		os.Getenv(name),
	)

	if value == "" {
		return defaultValue
	}

	return value
}

func abs(value int) int {
	if value < 0 {
		return -value
	}

	return value
}