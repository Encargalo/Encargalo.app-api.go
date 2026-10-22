package uploadimage

import (
	"errors"
	"testing"

	"Encargalo.app-api.go/internal/shared/errcustom"
)

func TestValidateImageDataURL(t *testing.T) {
	// 1x1 png real, el payload que mandan los clientes.
	const pngB64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFAAH/q842iQAAAABJRU5ErkJggg=="

	tests := []struct {
		name  string
		image string
		ok    bool
	}{
		{name: "png", image: "data:image/png;base64," + pngB64, ok: true},
		{name: "jpeg", image: "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEAYABgAAD/2wBD", ok: true},
		{name: "webp", image: "data:image/webp;base64," + pngB64, ok: true},
		{name: "heic", image: "data:image/heic;base64," + pngB64, ok: true},
		{name: "mayusculas en el prefijo", image: "DATA:IMAGE/PNG;BASE64," + pngB64, ok: true},
		{name: "con charset extra", image: "data:image/png;charset=utf-8;base64," + pngB64, ok: true},
		{name: "con espacios alrededor", image: "  data:image/png;base64," + pngB64 + "  ", ok: true},
		{name: "base64 con saltos de linea", image: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAAB\nCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFAAH/q842iQAAAABJRU5ErkJggg==", ok: true},
		{name: "base64 sin padding", image: "data:image/png;base64,iVBORw0KGgo", ok: true},

		{name: "url remota https", image: "https://attacker.example.com/payload.png", ok: false},
		{name: "url remota http", image: "http://169.254.169.254/latest/meta-data/", ok: false},
		{name: "ruta absoluta del servidor", image: "/etc/passwd", ok: false},
		{name: "ruta relativa del servidor", image: "../../etc/shadow", ok: false},
		{name: "ruta con coma", image: "/tmp/a,b/secret.env", ok: false},
		{name: "data-url de html", image: "data:text/html;base64,PHNjcmlwdD5hbGVydCgxKTwvc2NyaXB0Pg==", ok: false},
		{name: "data-url de svg", image: "data:image/svg+xml;base64,PHN2Zy8+", ok: false},
		{name: "data-url sin base64", image: "data:image/png,iVBORw0KGgo", ok: false},
		{name: "data-url sin coma", image: "data:image/png;base64", ok: false},
		{name: "payload vacio", image: "data:image/png;base64,", ok: false},
		{name: "payload no base64", image: "data:image/png;base64,no-es-base64-$$$", ok: false},
		{name: "string vacio", image: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateImageDataURL(tt.image)
			if tt.ok && err != nil {
				t.Fatalf("se esperaba válido, got %v", err)
			}
			if !tt.ok && !errors.Is(err, errcustom.ErrUploadingImage) {
				t.Fatalf("se esperaba ErrUploadingImage, got %v", err)
			}
		})
	}
}
