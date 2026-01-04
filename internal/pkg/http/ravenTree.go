package http

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"Encargalo.app-api.go/internal/shared/config"
	"github.com/AndresXLP/ravenTree"
)

type Http interface {
	RavenTree(ctx context.Context, path, method string, body []byte)
}

type httpImpl struct {
}

func NewHttp() Http {
	return &httpImpl{}
}

func (r *httpImpl) RavenTree(ctx context.Context, path, method string, body []byte) {
	config := config.Get().CreditManager

	tree := ravenTree.NewRavensTree()

	options := &ravenTree.Options{
		Host:       config.Url,
		Path:       path,
		Method:     http.MethodPost,
		Body:       body,
		Timeout:    5 * time.Second,
		RetryCount: 3,
	}

	resp, err := tree.SendRaven(ctx, options)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp.ParseBodyToString())
}
