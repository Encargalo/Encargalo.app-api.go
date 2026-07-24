package uploadimage

import "testing"

func TestPublicIDFromSecureURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
		ok   bool
	}{
		{
			name: "simple",
			url:  "https://res.cloudinary.com/demo/image/upload/v1312461204/sample.jpg",
			want: "sample",
			ok:   true,
		},
		{
			name: "folder",
			url:  "https://res.cloudinary.com/demo/image/upload/v1234/encargalo/riders/abc123.webp",
			want: "encargalo/riders/abc123",
			ok:   true,
		},
		{
			name: "with transform",
			url:  "https://res.cloudinary.com/demo/image/upload/c_fill,h_100,w_100/v1/folder/name.png",
			want: "folder/name",
			ok:   true,
		},
		{
			name: "invalid",
			url:  "https://example.com/foo.jpg",
			ok:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := PublicIDFromSecureURL(tt.url)
			if ok != tt.ok {
				t.Fatalf("ok=%v want %v", ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}
