package response_test

import (
	"net/http"
	"testing"

	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/response"
	"github.com/stretchr/testify/assert"
)

type sampleDTO struct {
	Name string `json:"name"`
}

func TestSend_WithoutHeader_IsPassthroughToJSON(t *testing.T) {
	cases := []struct {
		name string
		data any
	}{
		{"struct", sampleDTO{Name: "Carlos"}},
		{"slice", []sampleDTO{{Name: "a"}, {Name: "b"}}},
		{"string", "customer updated success"},
		{"map", map[string]string{"message": "session deleted success"}},
		{"nil", nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := buildhttptest.SetupControllerCase("/x", http.MethodGet, nil)
			errWant := want.Ctx.JSON(http.StatusOK, tc.data)
			assert.NoError(t, errWant)

			got := buildhttptest.SetupControllerCase("/x", http.MethodGet, nil)
			errGot := response.Send(got.Ctx, http.StatusOK, tc.data)
			assert.NoError(t, errGot)

			assert.Equal(t, want.Res.Code, got.Res.Code)
			assert.Equal(t, want.Res.Body.String(), got.Res.Body.String())
		})
	}
}

func TestSend_WithOtherHeaderValues_BehavesLikeV1(t *testing.T) {
	for _, v := range []string{"1", "", "v2", "V2", "22"} {
		t.Run("value="+v, func(t *testing.T) {
			want := buildhttptest.SetupControllerCase("/x", http.MethodGet, nil)
			_ = want.Ctx.JSON(http.StatusOK, sampleDTO{Name: "Carlos"})

			got := buildhttptest.SetupControllerCase("/x", http.MethodGet, nil)
			if v != "" {
				got.Req.Header.Set(response.VersionHeader, v)
			}
			_ = response.Send(got.Ctx, http.StatusOK, sampleDTO{Name: "Carlos"})

			assert.Equal(t, want.Res.Body.String(), got.Res.Body.String())
		})
	}
}

func TestSend_WithV2Header_WrapsInEnvelope(t *testing.T) {
	tc := buildhttptest.SetupControllerCase("/x", http.MethodGet, nil)
	tc.Req.Header.Set(response.VersionHeader, response.VersionV2)

	err := response.Send(tc.Ctx, http.StatusOK, sampleDTO{Name: "Carlos"})
	assert.NoError(t, err)
	assert.JSONEq(t, `{"data":{"name":"Carlos"},"error":null}`, tc.Res.Body.String())
}

func TestSendWithMeta_WithoutHeader_MetaIsDiscarded(t *testing.T) {
	want := buildhttptest.SetupControllerCase("/x", http.MethodGet, nil)
	_ = want.Ctx.JSON(http.StatusOK, sampleDTO{Name: "Carlos"})

	got := buildhttptest.SetupControllerCase("/x", http.MethodGet, nil)
	err := response.SendWithMeta(got.Ctx, http.StatusOK, sampleDTO{Name: "Carlos"}, map[string]any{"page": 1})
	assert.NoError(t, err)

	assert.Equal(t, want.Res.Body.String(), got.Res.Body.String())
}

func TestSendWithMeta_WithV2Header_IncludesMeta(t *testing.T) {
	tc := buildhttptest.SetupControllerCase("/x", http.MethodGet, nil)
	tc.Req.Header.Set(response.VersionHeader, response.VersionV2)

	err := response.SendWithMeta(tc.Ctx, http.StatusOK, sampleDTO{Name: "Carlos"}, map[string]any{"page": 1})
	assert.NoError(t, err)
	assert.JSONEq(t, `{"data":{"name":"Carlos"},"error":null,"meta":{"page":1}}`, tc.Res.Body.String())
}

func TestSendError_WithoutHeader_ReturnsV1BodyUnchanged(t *testing.T) {
	v1Body := map[string]any{"message": "faltan datos", "missing": []string{"logo", "banner"}}

	want := buildhttptest.SetupControllerCase("/x", http.MethodPost, nil)
	_ = want.Ctx.JSON(http.StatusUnprocessableEntity, v1Body)

	got := buildhttptest.SetupControllerCase("/x", http.MethodPost, nil)
	err := response.SendError(got.Ctx, http.StatusUnprocessableEntity, v1Body, "shop_onboarding_incomplete", "faltan datos", nil)
	assert.NoError(t, err)

	assert.Equal(t, want.Res.Body.String(), got.Res.Body.String())
}

func TestSendError_WithV2Header_ReturnsEnvelopeError(t *testing.T) {
	tc := buildhttptest.SetupControllerCase("/x", http.MethodPost, nil)
	tc.Req.Header.Set(response.VersionHeader, response.VersionV2)

	v1Body := map[string]any{"message": "faltan datos", "missing": []string{"logo", "banner"}}
	err := response.SendError(tc.Ctx, http.StatusUnprocessableEntity, v1Body,
		"shop_onboarding_incomplete", "faltan datos", map[string]any{"missing": []string{"logo", "banner"}})
	assert.NoError(t, err)

	assert.JSONEq(t, `{
		"data": null,
		"error": {
			"code": "shop_onboarding_incomplete",
			"message": "faltan datos",
			"details": {"missing": ["logo", "banner"]}
		}
	}`, tc.Res.Body.String())
}

func TestIsV2(t *testing.T) {
	tc := buildhttptest.SetupControllerCase("/x", http.MethodGet, nil)
	assert.False(t, response.IsV2(tc.Ctx))

	tc.Req.Header.Set(response.VersionHeader, response.VersionV2)
	assert.True(t, response.IsV2(tc.Ctx))
}
